package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/internal/domain/entity"
)

// ConsentLogRepository EULA/개인정보 동의 로그 적재 저장소.
type ConsentLogRepository interface {
	// CreateWithClientConsentID 멱등키(client_consent_id) 를 포함해 로그를 저장한다.
	// userID 가 nil 이면 user_id 는 NULL 로 저장(로그인 전 발생).
	CreateWithClientConsentID(ctx context.Context, userID *uuid.UUID, req *entity.CreateConsentLogRequest) (*entity.ConsentLog, error)
	// FindByClientConsentID 멱등성 검사용.
	FindByClientConsentID(ctx context.Context, clientConsentID string) (*entity.ConsentLog, error)
}

// ConsentLogItemStatus 로그 적재 결과 상태.
type ConsentLogItemStatus string

const (
	ConsentLogStatusAccepted  ConsentLogItemStatus = "accepted"
	ConsentLogStatusDuplicate ConsentLogItemStatus = "duplicate"
	ConsentLogStatusRejected  ConsentLogItemStatus = "rejected"
)

// ConsentLogRejectReason rejected 시 사유 코드. 결정론적 실패만 부여.
// policy_version 화이트리스트 검증 안 함(합의) → invalid_payload 만 사용.
type ConsentLogRejectReason string

const (
	ConsentLogReasonInvalidPayload ConsentLogRejectReason = "invalid_payload"
)

// ConsentLogResultItem 단건/batch 항목별 결과.
type ConsentLogResultItem struct {
	ClientConsentID string                 `json:"client_consent_id"`
	Status          ConsentLogItemStatus   `json:"status"`
	Reason          ConsentLogRejectReason `json:"reason,omitempty"`
}

// BatchConsentLogsResponse batch 엔드포인트 응답.
type BatchConsentLogsResponse struct {
	Results []ConsentLogResultItem `json:"results"`
}
