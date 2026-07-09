package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/domain/repository"
)

// EventLogUseCase Lobby Event 상태변경 로그 적재 유스케이스.
// 순수 관측 로그 — sanity/화이트리스트 검증 없음(합의 (a)), EXP/게임상태 미러링 없음.
type EventLogUseCase interface {
	Create(ctx context.Context, userID uuid.UUID, req *entity.CreateEventLogRequest) (*repository.EventLogResultItem, error)
	CreateBatch(ctx context.Context, userID uuid.UUID, req *entity.BatchEventLogsRequest) (*repository.BatchEventLogsResponse, error)
}

type eventLogUseCase struct {
	eventLogRepo repository.EventLogRepository
	logger       StructuredLogger
}

func NewEventLogUseCase(eventLogRepo repository.EventLogRepository, logger StructuredLogger) EventLogUseCase {
	return &eventLogUseCase{eventLogRepo: eventLogRepo, logger: logger}
}

// Create 단건 적재. 멱등키 = client_log_id.
func (u *eventLogUseCase) Create(ctx context.Context, userID uuid.UUID, req *entity.CreateEventLogRequest) (*repository.EventLogResultItem, error) {
	if req == nil {
		return nil, errors.New("nil request")
	}
	seen := make(map[string]struct{}, 1)
	result := u.processItem(ctx, userID, req, seen)
	return &result, nil
}

// CreateBatch /api/v1/events/log/batch 처리. 직렬로 항목별 처리.
func (u *eventLogUseCase) CreateBatch(ctx context.Context, userID uuid.UUID, req *entity.BatchEventLogsRequest) (*repository.BatchEventLogsResponse, error) {
	if req == nil || len(req.Logs) == 0 {
		return nil, errors.New("empty logs")
	}

	results := make([]repository.EventLogResultItem, 0, len(req.Logs))

	// 같은 batch 내 client_log_id 중복 감지 (첫 등장 통과, 두 번째부터 duplicate).
	seen := make(map[string]struct{}, len(req.Logs))

	for i := range req.Logs {
		item := &req.Logs[i]
		results = append(results, u.processItem(ctx, userID, item, seen))
	}

	return &repository.BatchEventLogsResponse{Results: results}, nil
}

// processItem 단건/batch 공통 항목 처리. record_batch.processBatchItem 의 멱등 로직 미러
// (sanity/EXP 단계 제거). 결정론적 실패만 rejected(invalid_payload).
func (u *eventLogUseCase) processItem(
	ctx context.Context,
	userID uuid.UUID,
	item *entity.CreateEventLogRequest,
	seen map[string]struct{},
) repository.EventLogResultItem {
	out := repository.EventLogResultItem{ClientLogID: item.ClientLogID}

	// 1. 같은 batch 내 중복 — 첫 등장은 통과, 두 번째부터 duplicate.
	if _, dup := seen[item.ClientLogID]; dup {
		out.Status = repository.EventLogStatusDuplicate
		return out
	}
	seen[item.ClientLogID] = struct{}{}

	// 2. 이미 DB 에 존재하는지 (= 멱등성 검사).
	if existing, err := u.eventLogRepo.FindByClientLogID(ctx, item.ClientLogID); err == nil && existing != nil {
		out.Status = repository.EventLogStatusDuplicate
		return out
	}

	// 3. 저장.
	if _, err := u.eventLogRepo.CreateWithClientLogID(ctx, userID, item.ClientLogID, item); err != nil {
		// UNIQUE violation (FindByClientLogID 와의 race) 이면 duplicate 로 폴백.
		if existing, ferr := u.eventLogRepo.FindByClientLogID(ctx, item.ClientLogID); ferr == nil && existing != nil {
			out.Status = repository.EventLogStatusDuplicate
			return out
		}
		// 그 외 저장 실패는 결정론적 실패로 간주 → invalid_payload.
		if u.logger != nil {
			u.logger.LogStructured("event_log_create_failed", map[string]interface{}{
				"client_log_id": item.ClientLogID,
				"user_id":       userID,
				"event_key":     item.EventKey,
				"error":         err.Error(),
			})
		}
		out.Status = repository.EventLogStatusRejected
		out.Reason = repository.EventLogReasonInvalidPayload
		return out
	}

	out.Status = repository.EventLogStatusAccepted
	return out
}
