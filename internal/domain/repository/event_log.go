package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/internal/domain/entity"
)

// EventLogRepository Event 상태변경 로그 적재 저장소.
type EventLogRepository interface {
	// CreateWithClientLogID 멱등키(client_log_id) 를 포함해 로그를 저장한다.
	CreateWithClientLogID(ctx context.Context, userID uuid.UUID, clientLogID string, req *entity.CreateEventLogRequest) (*entity.EventLog, error)
	// FindByClientLogID 멱등성 검사용.
	FindByClientLogID(ctx context.Context, clientLogID string) (*entity.EventLog, error)
}

// EventLogItemStatus 로그 적재 결과 상태.
type EventLogItemStatus string

const (
	EventLogStatusAccepted  EventLogItemStatus = "accepted"
	EventLogStatusDuplicate EventLogItemStatus = "duplicate"
	EventLogStatusRejected  EventLogItemStatus = "rejected"
)

// EventLogRejectReason rejected 시 사유 코드. 결정론적 실패만 부여 (재시도해도 같은 결과).
// event_key/state 화이트리스트 검증은 하지 않으므로(합의 (a)) invalid_payload 만 사용.
type EventLogRejectReason string

const (
	EventLogReasonInvalidPayload EventLogRejectReason = "invalid_payload"
)

// EventLogResultItem 단건/batch 항목별 결과.
type EventLogResultItem struct {
	ClientLogID string               `json:"client_log_id"`
	Status      EventLogItemStatus   `json:"status"`
	Reason      EventLogRejectReason `json:"reason,omitempty"`
}

// BatchEventLogsResponse batch 엔드포인트 응답.
// record batch 와 달리 user_after 없음 (게임상태 미러링 없는 순수 관측 로그).
type BatchEventLogsResponse struct {
	Results []EventLogResultItem `json:"results"`
}
