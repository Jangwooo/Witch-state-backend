package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/ent"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/domain/repository"
)

// SanityValidator per-item 검증 인터페이스. batch / 단건 양쪽에서 공유.
// 본체 구현은 Q7~Q10 합의 후 채운다 — 현재는 nil 또는 noopSanityValidator 주입 가능.
//
// stage / music 은 호출자가 이미 조회해서 넘긴다 (중복 조회 회피).
// stage / music 미존재는 Validate 호출 전에 호출자가 unknown_music/unknown_stage 로 처리.
type SanityValidator interface {
	Validate(req *entity.CreateRecordRequest, music *ent.Music, stage *ent.Stage) repository.BatchRejectReason
}

// noopSanityValidator: 모든 항목 통과. Q7~Q10 합의 전 임시.
type noopSanityValidator struct{}

func (noopSanityValidator) Validate(*entity.CreateRecordRequest, *ent.Music, *ent.Stage) repository.BatchRejectReason {
	return ""
}

// CreateBatch /api/v1/records/batch 처리.
// 직렬로 항목별 처리. 멱등 키 = client_record_id. user_after 는 마지막 accepted 의 exp_to 미러.
func (u *recordUseCase) CreateBatch(ctx context.Context, userID uuid.UUID, req *entity.BatchRecordsRequest) (*repository.BatchRecordsResponse, error) {
	if len(req.Records) == 0 {
		return nil, errors.New("empty records")
	}

	// batch 시작 시점 user 상태 — accepted 0건일 때 user_after 폴백.
	user, err := u.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	startExp, startLevel := user.Exp, user.Level

	results := make([]repository.BatchResultItem, 0, len(req.Records))
	userAfter := repository.BatchUserAfter{Exp: startExp, Level: startLevel}

	// 같은 batch 내 client_record_id 중복 감지 (정상 케이스 아니지만 두 번째부터 duplicate 처리)
	seen := make(map[string]struct{}, len(req.Records))

	for i := range req.Records {
		item := &req.Records[i]
		result := u.processBatchItem(ctx, userID, item, seen)
		results = append(results, result)

		// accepted 인 경우만 user_after 갱신 (Q4 결정 — 마지막 accepted 의 exp_to 미러)
		if result.Status == repository.BatchStatusAccepted && result.ExpGain != nil {
			userAfter.Exp = result.ExpGain.ExpTo
			userAfter.Level = result.ExpGain.LevelTo
		}
	}

	return &repository.BatchRecordsResponse{
		Results:   results,
		UserAfter: userAfter,
	}, nil
}

func (u *recordUseCase) processBatchItem(
	ctx context.Context,
	userID uuid.UUID,
	item *entity.BatchRecordItem,
	seen map[string]struct{},
) repository.BatchResultItem {
	out := repository.BatchResultItem{
		ClientRecordID: item.ClientRecordID,
	}

	// 1. 같은 batch 내 중복 — 첫 등장은 통과, 두 번째부터 duplicate.
	if _, dup := seen[item.ClientRecordID]; dup {
		// 같은 batch 안 중복은 비정상. 기존 record_id 도 모르므로 nil.
		out.Status = repository.BatchStatusDuplicate
		return out
	}
	seen[item.ClientRecordID] = struct{}{}

	// 2. 이미 DB 에 존재하는지 (= 멱등성 검사)
	if existing, err := u.recordRepo.FindByClientRecordID(ctx, item.ClientRecordID); err == nil && existing != nil {
		out.Status = repository.BatchStatusDuplicate
		rid := existing.ID
		out.RecordID = &rid
		// Q6: payload mismatch 시 warn 로그. 거부 안 함.
		u.logDuplicatePayloadMismatch(existing, item)
		return out
	}

	// 3. music / stage 존재 확인 + sanity validator.
	//    SANITY_MODE = off    → 모두 스킵 (record 그대로 저장 시도; FK 위반 시 5단계에서 invalid_payload 분류)
	//                  shadow → 위반 발견 시 로그만 남기고 통과
	//                  enforce→ 위반 시 rejected
	if u.sanityMode != SanityModeOff {
		music, err := u.musicRepo.GetMusicByID(ctx, item.MusicID)
		if err != nil || music == nil {
			if u.sanityMode == SanityModeEnforce {
				out.Status = repository.BatchStatusRejected
				out.Reason = repository.ReasonUnknownMusic
				return out
			}
			u.logSanityShadowEntityMiss(&item.CreateRecordRequest, repository.ReasonUnknownMusic)
		} else {
			stage, err := u.stageRepo.GetStageByID(ctx, item.StageID)
			if err != nil || stage == nil {
				if u.sanityMode == SanityModeEnforce {
					out.Status = repository.BatchStatusRejected
					out.Reason = repository.ReasonUnknownStage
					return out
				}
				u.logSanityShadowEntityMiss(&item.CreateRecordRequest, repository.ReasonUnknownStage)
			} else if u.sanityValidator != nil {
				// sanityValidator 자체가 shadow/enforce 분기 내장 (sanity_validator.go).
				if reason := u.sanityValidator.Validate(&item.CreateRecordRequest, music, stage); reason != "" {
					out.Status = repository.BatchStatusRejected
					out.Reason = reason
					return out
				}
			}
		}
	}

	// 5. 저장
	created, err := u.recordRepo.CreateWithClientRecordID(ctx, userID, item.ClientRecordID, &item.CreateRecordRequest)
	if err != nil {
		// UNIQUE violation 으로 떨어졌다면 (FindByClientRecordID 와의 race) duplicate 로 폴백
		if existing, ferr := u.recordRepo.FindByClientRecordID(ctx, item.ClientRecordID); ferr == nil && existing != nil {
			out.Status = repository.BatchStatusDuplicate
			rid := existing.ID
			out.RecordID = &rid
			u.logDuplicatePayloadMismatch(existing, item)
			return out
		}
		// 그 외 저장 실패는 invalid_payload 로 분류 (결정론적 실패만 reject — 일시 장애는 핸들러에서 5xx 로 분리해야 하지만,
		// per-item 단위에선 결정론적이라 가정. 일시 장애가 의심되면 추후 5xx 로 escalate)
		out.Status = repository.BatchStatusRejected
		out.Reason = repository.ReasonInvalidPayload
		return out
	}
	rid := created.ID
	out.Status = repository.BatchStatusAccepted
	out.RecordID = &rid

	// 6. EXP 재계산 — completed 항목만 (기존 단건 규약과 일치)
	if item.GameStatus == "" || item.GameStatus == "completed" {
		gain, err := u.applyExpGain(ctx, userID, &item.CreateRecordRequest)
		if err == nil && gain != nil {
			out.ExpGain = gain
		}
	}

	return out
}

// logDuplicatePayloadMismatch Q6: 같은 client_record_id 인데 페이로드가 다르면 warn 로그.
// 거부는 안 함 — 클라 큐가 영구히 막히는 것을 피한다. 위조 추적은 로그 분석으로.
func (u *recordUseCase) logDuplicatePayloadMismatch(existing *entity.Record, incoming *entity.BatchRecordItem) {
	if u.logger == nil {
		return
	}

	diff := map[string]interface{}{}
	if existing.Score != incoming.Score {
		diff["score"] = [2]int{existing.Score, incoming.Score}
	}
	if existing.PerfectCount != incoming.PerfectCount {
		diff["perfect_count"] = [2]int{existing.PerfectCount, incoming.PerfectCount}
	}
	if existing.GoodCount != incoming.GoodCount {
		diff["good_count"] = [2]int{existing.GoodCount, incoming.GoodCount}
	}
	if existing.BadCount != incoming.BadCount {
		diff["bad_count"] = [2]int{existing.BadCount, incoming.BadCount}
	}
	if existing.MissCount != incoming.MissCount {
		diff["miss_count"] = [2]int{existing.MissCount, incoming.MissCount}
	}
	if existing.MaxCombo != incoming.MaxCombo {
		diff["max_combo"] = [2]int{existing.MaxCombo, incoming.MaxCombo}
	}
	if existing.Accuracy != incoming.Accuracy {
		diff["accuracy"] = [2]float64{existing.Accuracy, incoming.Accuracy}
	}
	if string(existing.Rank) != incoming.Rank {
		diff["rank"] = [2]string{string(existing.Rank), incoming.Rank}
	}
	if len(diff) == 0 {
		return // 동일 페이로드 — 로그 불필요
	}

	u.logger.LogStructured("batch_duplicate_payload_mismatch", map[string]interface{}{
		"client_record_id": incoming.ClientRecordID,
		"record_id":        existing.ID,
		"user_id":          existing.UserID,
		"music_id":         incoming.MusicID,
		"stage_id":         incoming.StageID,
		"diff":             diff,
	})
}
