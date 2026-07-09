package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/ent"
)

// ConsentLog 도메인 엔티티 래퍼
type ConsentLog struct {
	*ent.ConsentLog
}

// CreateConsentLogRequest 단건 /api/v1/consents/log 요청 본문.
// user_id 는 body 에 없다 — 로그인 전 발생. 세션(Bearer)이 있으면 서버가 채우고, 없으면 null.
type CreateConsentLogRequest struct {
	ClientConsentID string `json:"client_consent_id" validate:"required,uuid4"`
	ClientID        string `json:"client_id" validate:"required"`
	ConsentType     string `json:"consent_type" validate:"required,oneof=eula privacy"`
	PolicyVersion   string `json:"policy_version" validate:"required"`
	Granted         bool   `json:"granted"`
	Nickname        string `json:"nickname"`
	ConsentedAt     int64  `json:"consented_at" validate:"required"` // unix seconds (UTC)
	ClientVersion   string `json:"client_version"`
	Platform        string `json:"platform"`
	Locale          string `json:"locale"`
	BuildMode       string `json:"build_mode"`
}

// BatchConsentLogsRequest /api/v1/consents/log/batch 요청 본문.
type BatchConsentLogsRequest struct {
	Consents []CreateConsentLogRequest `json:"consents" validate:"required,min=1,max=100,dive"`
}

// ConsentedAtTime 은 클라 전송 unix seconds 를 UTC time.Time 으로 변환한다.
func (r *CreateConsentLogRequest) ConsentedAtTime() time.Time {
	return time.Unix(r.ConsentedAt, 0).UTC()
}

func NewConsentLog(entLog *ent.ConsentLog) *ConsentLog {
	return &ConsentLog{ConsentLog: entLog}
}

// ConsentLogResponse 적재된 동의 로그 표현 (필요 시 조회용).
type ConsentLogResponse struct {
	ID              uuid.UUID  `json:"id"`
	ClientConsentID uuid.UUID  `json:"client_consent_id"`
	ClientID        uuid.UUID  `json:"client_id"`
	UserID          *uuid.UUID `json:"user_id,omitempty"`
	ConsentType     string     `json:"consent_type"`
	PolicyVersion   string     `json:"policy_version"`
	Granted         bool       `json:"granted"`
	Nickname        string     `json:"nickname,omitempty"`
	ConsentedAt     time.Time  `json:"consented_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

func (l *ConsentLog) ToResponse() ConsentLogResponse {
	return ConsentLogResponse{
		ID:              l.ID,
		ClientConsentID: l.ClientConsentID,
		ClientID:        l.ClientID,
		UserID:          l.UserID,
		ConsentType:     string(l.ConsentType),
		PolicyVersion:   l.PolicyVersion,
		Granted:         l.Granted,
		Nickname:        l.Nickname,
		ConsentedAt:     l.ConsentedAt,
		CreatedAt:       l.CreatedAt,
	}
}
