package usecase

import (
	"context"
	"math"

	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/domain/repository"
	infrarepo "github.com/witchs-lounge_backend/internal/repository"
)

type RecordUseCase interface {
	Create(ctx context.Context, userID uuid.UUID, req *entity.CreateRecordRequest) (*repository.SingleRecordResponse, error)
	CreateBatch(ctx context.Context, userID uuid.UUID, req *entity.BatchRecordsRequest) (*repository.BatchRecordsResponse, error)
	List(ctx context.Context, userID uuid.UUID, musicID, stageID string) (*repository.ListRecordResponse, error)
	Best(ctx context.Context, userID uuid.UUID, musicID, stageID string) (*repository.BestRecordResponse, error)
}

// StructuredLogger usecase 가 logging 패키지에 직접 의존하지 않도록 추출한 최소 인터페이스.
type StructuredLogger interface {
	LogStructured(eventType string, fields map[string]interface{})
}

type recordUseCase struct {
	recordRepo      repository.RecordRepository
	userRepo        repository.UserRepository
	musicRepo       infrarepo.MusicRepository
	stageRepo       infrarepo.StageRepository
	progressionRepo infrarepo.ProgressionRepository
	sanityValidator SanityValidator
	sanityMode      SanityMode
	logger          StructuredLogger
}

func NewRecordUseCase(
	recordRepo repository.RecordRepository,
	userRepo repository.UserRepository,
	musicRepo infrarepo.MusicRepository,
	stageRepo infrarepo.StageRepository,
	progressionRepo infrarepo.ProgressionRepository,
	sanityValidator SanityValidator,
	sanityMode SanityMode,
	logger StructuredLogger,
) RecordUseCase {
	if sanityValidator == nil {
		sanityValidator = noopSanityValidator{}
	}
	return &recordUseCase{
		recordRepo:      recordRepo,
		userRepo:        userRepo,
		musicRepo:       musicRepo,
		stageRepo:       stageRepo,
		progressionRepo: progressionRepo,
		sanityValidator: sanityValidator,
		sanityMode:      sanityMode,
		logger:          logger,
	}
}

// SanityRejectError sanity validator reject 시 반환되는 sentinel error.
// handler 가 errors.As 로 잡아 400 + ErrorResponse 응답으로 변환한다 (Q12 옵션 A).
type SanityRejectError struct {
	Reason repository.BatchRejectReason
}

func (e *SanityRejectError) Error() string {
	return "sanity_rejected: " + string(e.Reason)
}

func (u *recordUseCase) Create(ctx context.Context, userID uuid.UUID, req *entity.CreateRecordRequest) (*repository.SingleRecordResponse, error) {
	// SANITY_MODE 분기:
	//   off    → music/stage 조회·검증 모두 스킵 (기존 운영 동작과 동일, 100% 호환)
	//   shadow → 검증 실행 + 위반 로그만, reject 안 함 (200 통과)
	//   enforce→ 검증 실행, music/stage/sanity 위반 시 400
	if u.sanityMode != SanityModeOff {
		music, mErr := u.musicRepo.GetMusicByID(ctx, req.MusicID)
		if mErr != nil || music == nil {
			if u.sanityMode == SanityModeEnforce {
				return nil, &SanityRejectError{Reason: repository.ReasonUnknownMusic}
			}
			u.logSanityShadowEntityMiss(req, repository.ReasonUnknownMusic)
		} else {
			stage, sErr := u.stageRepo.GetStageByID(ctx, req.StageID)
			if sErr != nil || stage == nil {
				if u.sanityMode == SanityModeEnforce {
					return nil, &SanityRejectError{Reason: repository.ReasonUnknownStage}
				}
				u.logSanityShadowEntityMiss(req, repository.ReasonUnknownStage)
			} else if u.sanityValidator != nil {
				// sanityValidator 자체가 shadow/enforce 분기 내장 (sanity_validator.go).
				if reason := u.sanityValidator.Validate(req, music, stage); reason != "" {
					return nil, &SanityRejectError{Reason: reason}
				}
			}
		}
	}

	created, err := u.recordRepo.Create(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	rr := toRecordResponse(created)
	resp := &repository.SingleRecordResponse{Record: rr}

	// Complete 상태인 기록만 경험치로 인정
	if req.GameStatus == "" || req.GameStatus == "completed" {
		gain, err := u.applyExpGain(ctx, userID, req)
		if err == nil && gain != nil {
			resp.ExpGain = gain
		}
	}

	return resp, nil
}

// logSanityShadowEntityMiss: shadow 모드에서 music/stage 미존재를 발견했을 때 로그.
// reject 하지 않고 통과시키되 운영자가 enforce 전환 전 위반율 추적 가능.
func (u *recordUseCase) logSanityShadowEntityMiss(req *entity.CreateRecordRequest, reason repository.BatchRejectReason) {
	if u.logger == nil {
		return
	}
	u.logger.LogStructured("sanity_shadow_violation", map[string]interface{}{
		"reason":   string(reason),
		"music_id": req.MusicID,
		"stage_id": req.StageID,
	})
}

// applyExpGain 계산식:
//   (1 + 0.0001 * notes) * (1 + 0.01 * difficulty) * rank_coefficient
//   * (miss==0 ? 1.05 : 1) * (all perfect ? 1.1 : 1) * 100
// 노트 수는 판정 합계(perfect+good+bad+miss)로 산출.
// 한 플레이는 "miss 없음" 와 "all perfect" 가 동시에 성립할 수 있는데,
// 명세상 "miss 없으면 1.05 / 전부 perfect 면 1.1 / 그 외 1" 로 한 가지만 적용됨.
func (u *recordUseCase) applyExpGain(ctx context.Context, userID uuid.UUID, req *entity.CreateRecordRequest) (*repository.ExpGain, error) {
	stage, err := u.stageRepo.GetStageByID(ctx, req.StageID)
	if err != nil {
		return nil, err
	}

	coeff, err := u.progressionRepo.GetRankCoefficient(ctx, req.Rank)
	if err != nil {
		return nil, err
	}

	notes := req.PerfectCount + req.GoodCount + req.BadCount + req.MissCount

	var judgmentMul float64 = 1.0
	switch {
	case req.GoodCount == 0 && req.BadCount == 0 && req.MissCount == 0 && req.PerfectCount > 0:
		judgmentMul = 1.1
	case req.MissCount == 0:
		judgmentMul = 1.05
	}

	exp := (1.0 + 0.0001*float64(notes)) *
		(1.0 + 0.01*float64(stage.Difficulty)) *
		float64(coeff) *
		judgmentMul *
		100.0

	gained := int(math.Floor(exp))
	if gained <= 0 {
		return nil, nil
	}

	user, err := u.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	expFrom := user.Exp
	levelFrom := user.Level
	expTo := expFrom + gained

	levelTo, err := u.progressionRepo.GetHighestLevelForExp(ctx, expTo)
	if err != nil {
		// t_level 조회 실패 시 레벨은 유지
		levelTo = levelFrom
	}
	if levelTo < levelFrom {
		levelTo = levelFrom
	}

	if _, err := u.userRepo.UpdateExpAndLevel(ctx, userID, expTo, levelTo); err != nil {
		return nil, err
	}

	return &repository.ExpGain{
		ExpFrom:   expFrom,
		ExpTo:     expTo,
		ExpGained: gained,
		LevelFrom: levelFrom,
		LevelTo:   levelTo,
	}, nil
}

func (u *recordUseCase) List(ctx context.Context, userID uuid.UUID, musicID, stageID string) (*repository.ListRecordResponse, error) {
	list, err := u.recordRepo.ListByUserAndMusicStage(ctx, userID, musicID, stageID)
	if err != nil {
		return nil, err
	}
	out := make([]repository.RecordResponse, 0, len(list))
	for _, v := range list {
		out = append(out, toRecordResponse(v))
	}
	return &repository.ListRecordResponse{Records: out}, nil
}

func (u *recordUseCase) Best(ctx context.Context, userID uuid.UUID, musicID, stageID string) (*repository.BestRecordResponse, error) {
	best, err := u.recordRepo.BestByUserAndMusicStage(ctx, userID, musicID, stageID)
	if err != nil {
		return &repository.BestRecordResponse{Record: nil}, nil
	}
	if best == nil {
		return &repository.BestRecordResponse{Record: nil}, nil
	}
	br := toRecordResponse(best)
	return &repository.BestRecordResponse{Record: &br}, nil
}

// toRecordResponse: entity.Record -> repository.RecordResponse
func toRecordResponse(r *entity.Record) repository.RecordResponse {
	return repository.RecordResponse{
		ID:            r.ID,
		UserID:        r.UserID,
		MusicID:       r.MusicID,
		StageID:       r.StageID,
		Score:         r.Score,
		PerfectCount:  r.PerfectCount,
		GoodCount:     r.GoodCount,
		BadCount:      r.BadCount,
		MissCount:     r.MissCount,
		MaxCombo:      r.MaxCombo,
		Accuracy:      r.Accuracy,
		Rank:          string(r.Rank),
		IsFullCombo:   r.IsFullCombo,
		IsPerfectPlay: r.IsPerfectPlay,
		Additional:    r.AdditionalInfo,
		CreatedAt:     r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     r.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
