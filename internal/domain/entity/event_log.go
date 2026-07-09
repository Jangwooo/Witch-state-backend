package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/ent"
)

// EventLog 도메인 엔티티 래퍼
type EventLog struct {
	*ent.EventLog
}

// CreateEventLogRequest 단건 /api/v1/events/log 요청 본문.
// userId 는 body 에 없음 — 서버가 세션에서 도출한다.
type CreateEventLogRequest struct {
	ClientLogID string `json:"client_log_id" validate:"required,uuid4"`
	EventKey    string `json:"event_key" validate:"required"`
	StateBefore string `json:"state_before" validate:"required"`
	StateAfter  string `json:"state_after" validate:"required"`
	ChangedAt   int64  `json:"changed_at" validate:"required"` // 클라 발생 시각 (unix seconds)
}

// BatchEventLogsRequest /api/v1/events/log/batch 요청 본문.
type BatchEventLogsRequest struct {
	Logs []CreateEventLogRequest `json:"logs" validate:"required,min=1,max=100,dive"`
}

// ChangedAtTime 은 클라 전송 unix seconds 를 UTC time.Time 으로 변환한다.
func (r *CreateEventLogRequest) ChangedAtTime() time.Time {
	return time.Unix(r.ChangedAt, 0).UTC()
}

func NewEventLog(entLog *ent.EventLog) *EventLog {
	return &EventLog{EventLog: entLog}
}

// EventLogResponse 적재된 로그 표현 (필요 시 조회용).
type EventLogResponse struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	EventKey    string    `json:"event_key"`
	StateBefore string    `json:"state_before"`
	StateAfter  string    `json:"state_after"`
	ChangedAt   time.Time `json:"changed_at"`
	CreatedAt   time.Time `json:"created_at"`
}

func (l *EventLog) ToResponse() EventLogResponse {
	return EventLogResponse{
		ID:          l.ID,
		UserID:      l.UserID,
		EventKey:    l.EventKey,
		StateBefore: l.StateBefore,
		StateAfter:  l.StateAfter,
		ChangedAt:   l.ChangedAt,
		CreatedAt:   l.CreatedAt,
	}
}
