package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/domain/repository"
)

// ConsentLogUseCase EULA/개인정보 동의 로그 적재 유스케이스.
// 무인증 라우트 — userID 는 세션(Bearer)이 있을 때만 채워지고, 없으면 nil.
// policy_version 화이트리스트 검증 없음, EXP/게임상태 미러링 없음.
type ConsentLogUseCase interface {
	Create(ctx context.Context, userID *uuid.UUID, req *entity.CreateConsentLogRequest) (*repository.ConsentLogResultItem, error)
	CreateBatch(ctx context.Context, userID *uuid.UUID, req *entity.BatchConsentLogsRequest) (*repository.BatchConsentLogsResponse, error)
}

type consentLogUseCase struct {
	consentLogRepo repository.ConsentLogRepository
	logger         StructuredLogger
}

func NewConsentLogUseCase(consentLogRepo repository.ConsentLogRepository, logger StructuredLogger) ConsentLogUseCase {
	return &consentLogUseCase{consentLogRepo: consentLogRepo, logger: logger}
}

// Create 단건 적재. 멱등키 = client_consent_id.
func (u *consentLogUseCase) Create(ctx context.Context, userID *uuid.UUID, req *entity.CreateConsentLogRequest) (*repository.ConsentLogResultItem, error) {
	if req == nil {
		return nil, errors.New("nil request")
	}
	seen := make(map[string]struct{}, 1)
	result := u.processItem(ctx, userID, req, seen)
	return &result, nil
}

// CreateBatch /api/v1/consents/log/batch 처리. 직렬로 항목별 처리.
func (u *consentLogUseCase) CreateBatch(ctx context.Context, userID *uuid.UUID, req *entity.BatchConsentLogsRequest) (*repository.BatchConsentLogsResponse, error) {
	if req == nil || len(req.Consents) == 0 {
		return nil, errors.New("empty consents")
	}

	results := make([]repository.ConsentLogResultItem, 0, len(req.Consents))

	// 같은 batch 내 client_consent_id 중복 감지 (첫 등장 통과, 두 번째부터 duplicate).
	seen := make(map[string]struct{}, len(req.Consents))

	for i := range req.Consents {
		item := &req.Consents[i]
		results = append(results, u.processItem(ctx, userID, item, seen))
	}

	return &repository.BatchConsentLogsResponse{Results: results}, nil
}

// processItem 단건/batch 공통 항목 처리. event_log 멱등 로직 미러 (user_id nullable).
func (u *consentLogUseCase) processItem(
	ctx context.Context,
	userID *uuid.UUID,
	item *entity.CreateConsentLogRequest,
	seen map[string]struct{},
) repository.ConsentLogResultItem {
	out := repository.ConsentLogResultItem{ClientConsentID: item.ClientConsentID}

	// 1. 같은 batch 내 중복 — 첫 등장은 통과, 두 번째부터 duplicate.
	if _, dup := seen[item.ClientConsentID]; dup {
		out.Status = repository.ConsentLogStatusDuplicate
		return out
	}
	seen[item.ClientConsentID] = struct{}{}

	// 2. 이미 DB 에 존재하는지 (= 멱등성 검사).
	if existing, err := u.consentLogRepo.FindByClientConsentID(ctx, item.ClientConsentID); err == nil && existing != nil {
		out.Status = repository.ConsentLogStatusDuplicate
		return out
	}

	// 3. 저장.
	if _, err := u.consentLogRepo.CreateWithClientConsentID(ctx, userID, item); err != nil {
		// UNIQUE violation (FindByClientConsentID 와의 race) 이면 duplicate 로 폴백.
		if existing, ferr := u.consentLogRepo.FindByClientConsentID(ctx, item.ClientConsentID); ferr == nil && existing != nil {
			out.Status = repository.ConsentLogStatusDuplicate
			return out
		}
		// 그 외 저장 실패는 결정론적 실패로 간주 → invalid_payload.
		if u.logger != nil {
			u.logger.LogStructured("consent_log_create_failed", map[string]interface{}{
				"client_consent_id": item.ClientConsentID,
				"client_id":         item.ClientID,
				"consent_type":      item.ConsentType,
				"error":             err.Error(),
			})
		}
		out.Status = repository.ConsentLogStatusRejected
		out.Reason = repository.ConsentLogReasonInvalidPayload
		return out
	}

	out.Status = repository.ConsentLogStatusAccepted
	return out
}
