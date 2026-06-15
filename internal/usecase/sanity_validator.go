package usecase

import (
	"math"
	"strings"

	"github.com/spf13/viper"
	"github.com/witchs-lounge_backend/ent"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/domain/repository"
)

// SanityMode 는 sanity validator 의 운영 모드. HMAC_MODE 와 동일한 단계적 롤아웃 패턴.
//   - SanityModeOff: validator 우회. record 가 그대로 저장 시도 (기존 운영 동작과 동일). 100% 호환.
//   - SanityModeShadow: validator 실행. 위반 시 구조화 로그만 남기고 reject 하지 않음 (200 통과).
//   - SanityModeEnforce: validator 실행. 위반 시 reject (단건 400 / batch 항목별 rejected).
type SanityMode string

const (
	SanityModeOff     SanityMode = "off"
	SanityModeShadow  SanityMode = "shadow"
	SanityModeEnforce SanityMode = "enforce"
)

// LoadSanityMode 는 SANITY_MODE 환경변수에서 운영 모드를 읽는다. 미설정/오타 시 off.
func LoadSanityMode() SanityMode {
	switch strings.ToLower(strings.TrimSpace(viper.GetString("SANITY_MODE"))) {
	case "shadow":
		return SanityModeShadow
	case "enforce":
		return SanityModeEnforce
	default:
		return SanityModeOff
	}
}

// 클라 산식 미러 상수 (Q7-c). 변경 시 client (StageConstant.cs) 와 동기화 필요.
const (
	scorePerfect = 1000
	scoreGood    = 500
	scoreBad     = 200

	accuracyWeightPerfect = 10000
	accuracyWeightGood    = 6600
	accuracyWeightBad     = 3300

	// Q8 합의된 duration 가드 비율.
	durationLowerRatio = 0.5
	durationUpperRatio = 1.1

	// Q7 accuracy 검증 허용오차 (정수 나눗셈 1 단위 = ±0.01%p).
	accuracyTolerance = 1
)

// defaultSanityValidator Q7~Q10 합의된 per-item 검증.
// batch / 단건 양쪽에서 같은 인스턴스 사용.
//
// mode == off 면 Validate 가 무조건 빈 reason 반환 (기존 운영 동작과 동일).
// mode == shadow 면 위반을 logger 로 남기고 빈 reason 반환 (200 통과).
// mode == enforce 면 위반 시 reason 반환 (단건 400 / batch rejected).
type defaultSanityValidator struct {
	mode   SanityMode
	logger StructuredLogger
}

// NewSanityValidator returns the production validator.
// mode == off 인 경우에도 같은 타입 반환 — Validate 호출 시점에 분기.
// logger 가 nil 이면 shadow 위반 로그는 silent.
func NewSanityValidator(mode SanityMode, logger StructuredLogger) SanityValidator {
	return &defaultSanityValidator{mode: mode, logger: logger}
}

func (v *defaultSanityValidator) Validate(req *entity.CreateRecordRequest, music *ent.Music, stage *ent.Stage) repository.BatchRejectReason {
	if v.mode == SanityModeOff {
		return ""
	}

	reason := v.evaluate(req, music)

	if reason == "" {
		return ""
	}

	if v.mode == SanityModeShadow {
		v.logShadowViolation(req, music, stage, reason)
		return ""
	}

	// enforce
	return reason
}

// logShadowViolation: shadow 모드에서 위반 발생 시 구조화 로그.
// reject 하지 않으므로 운영자가 enforce 전환 전 위반율을 모니터링 가능.
func (v *defaultSanityValidator) logShadowViolation(req *entity.CreateRecordRequest, music *ent.Music, stage *ent.Stage, reason repository.BatchRejectReason) {
	if v.logger == nil {
		return
	}
	fields := map[string]interface{}{
		"reason":        string(reason),
		"music_id":      req.MusicID,
		"stage_id":      req.StageID,
		"score":         req.Score,
		"perfect_count": req.PerfectCount,
		"good_count":    req.GoodCount,
		"bad_count":     req.BadCount,
		"miss_count":    req.MissCount,
		"accuracy":      req.Accuracy,
		"play_duration": req.PlayDuration,
	}
	if music != nil {
		fields["music_duration_seconds"] = music.DurationSeconds
	}
	if stage != nil {
		fields["stage_difficulty"] = stage.Difficulty
	}
	v.logger.LogStructured("sanity_shadow_violation", fields)
}

// evaluate: 실제 검증 로직. mode 분기와 무관. enforce/shadow 양쪽이 호출.
func (v *defaultSanityValidator) evaluate(req *entity.CreateRecordRequest, music *ent.Music) repository.BatchRejectReason {
	// 0. 노트 수 = 판정 합계. 0 이하면 invalid (현 단계 약검증; Q7-a 결정).
	notes := req.PerfectCount + req.GoodCount + req.BadCount + req.MissCount
	if notes <= 0 {
		return repository.ReasonInvalidPayload
	}

	// 1. score 엄밀 검증 — Q9 / Q7-c: score == P*1000 + G*500 + B*200, miss 가산 없음.
	expectedScore := req.PerfectCount*scorePerfect + req.GoodCount*scoreGood + req.BadCount*scoreBad
	if req.Score != expectedScore {
		return repository.ReasonScoreOutOfRange
	}

	// 2. accuracy 정합성 — Q7-b: 클라 percent 산식 미러.
	//   expected = (P*10000 + G*6600 + B*3300) / (P+G+B+M)  (정수 나눗셈)
	//   |accuracy - expected| <= 1 (±0.01%p 허용오차).
	expectedPercent := (req.PerfectCount*accuracyWeightPerfect +
		req.GoodCount*accuracyWeightGood +
		req.BadCount*accuracyWeightBad) / notes

	// accuracy 는 ent schema 상 float64 이지만 단위는 int 0~10000 (Q11). 비교는 정수 의미로.
	actual := int(math.Round(req.Accuracy))
	if abs(actual-expectedPercent) > accuracyTolerance {
		return repository.ReasonPayloadInconsistent
	}

	// 3. duration 가드 — Q8: Music.duration_seconds 가 운영 데이터로 채워진 경우만 검증 (fail-open).
	//    duration_seconds == 0 → 운영자 미입력 → 검증 스킵.
	//    클라가 play_duration 을 보내지 않은 (== 0) 경우도 검증 스킵 (필드 미사용 시).
	if music != nil && music.DurationSeconds > 0 && req.PlayDuration > 0 {
		lower := music.DurationSeconds * durationLowerRatio
		upper := music.DurationSeconds * durationUpperRatio
		if req.PlayDuration < lower || req.PlayDuration > upper {
			return repository.ReasonDurationOutOfRange
		}
	}

	return ""
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
