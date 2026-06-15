package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/internal/domain/entity"
)

type RecordRepository interface {
	Create(ctx context.Context, userID uuid.UUID, req *entity.CreateRecordRequest) (*entity.Record, error)
	CreateWithClientRecordID(ctx context.Context, userID uuid.UUID, clientRecordID string, req *entity.CreateRecordRequest) (*entity.Record, error)
	FindByClientRecordID(ctx context.Context, clientRecordID string) (*entity.Record, error)
	ListByUserAndMusicStage(ctx context.Context, userID uuid.UUID, musicID, stageID string) ([]*entity.Record, error)
	BestByUserAndMusicStage(ctx context.Context, userID uuid.UUID, musicID, stageID string) (*entity.Record, error)
}

// Record 응답 타입들(요청 요구에 따라 repository 도메인에 정의)
type RecordResponse struct {
	ID            uuid.UUID              `json:"id"`
	UserID        uuid.UUID              `json:"user_id"`
	MusicID       string                 `json:"music_id"`
	StageID       string                 `json:"stage_id"`
	Score         int                    `json:"score"`
	PerfectCount  int                    `json:"perfect_count"`
	GoodCount     int                    `json:"good_count"`
	BadCount      int                    `json:"bad_count"`
	MissCount     int                    `json:"miss_count"`
	MaxCombo      int                    `json:"max_combo"`
	Accuracy      float64                `json:"accuracy"`
	Rank          string                 `json:"rank"`
	IsFullCombo   bool                   `json:"is_full_combo"`
	IsPerfectPlay bool                   `json:"is_perfect_play"`
	Additional    map[string]interface{} `json:"additional_info,omitempty"`
	CreatedAt     string                 `json:"created_at"`
	UpdatedAt     string                 `json:"updated_at"`
}

// ExpGain 기록 저장 시 경험치/레벨 변화 정보
type ExpGain struct {
	ExpFrom   int `json:"exp_from"`
	ExpTo     int `json:"exp_to"`
	ExpGained int `json:"exp_gained"`
	LevelFrom int `json:"level_from"`
	LevelTo   int `json:"level_to"`
}

type SingleRecordResponse struct {
	Record  RecordResponse `json:"record"`
	ExpGain *ExpGain       `json:"exp_gain,omitempty"`
}

type ListRecordResponse struct {
	Records []RecordResponse `json:"records"`
}

type BestRecordResponse struct {
	Record *RecordResponse `json:"record"`
}

// BatchItemStatus batch 항목 결과 상태
type BatchItemStatus string

const (
	BatchStatusAccepted  BatchItemStatus = "accepted"
	BatchStatusDuplicate BatchItemStatus = "duplicate"
	BatchStatusRejected  BatchItemStatus = "rejected"
)

// BatchRejectReason rejected 시 사유 코드. 결정론적 실패만 부여 (재시도해도 같은 결과).
type BatchRejectReason string

const (
	ReasonInvalidPayload      BatchRejectReason = "invalid_payload"
	ReasonUnknownMusic        BatchRejectReason = "unknown_music"
	ReasonUnknownStage        BatchRejectReason = "unknown_stage"
	ReasonScoreOutOfRange     BatchRejectReason = "score_out_of_range"
	ReasonPayloadInconsistent BatchRejectReason = "payload_inconsistent"
	ReasonDurationOutOfRange  BatchRejectReason = "duration_out_of_range"
)

// BatchResultItem batch 응답의 항목별 결과
type BatchResultItem struct {
	ClientRecordID string            `json:"client_record_id"`
	Status         BatchItemStatus   `json:"status"`
	RecordID       *uuid.UUID        `json:"record_id,omitempty"`
	ExpGain        *ExpGain          `json:"exp_gain,omitempty"`
	Reason         BatchRejectReason `json:"reason,omitempty"`
}

// BatchUserAfter batch 처리 후 유저 누적 상태
type BatchUserAfter struct {
	Exp   int `json:"exp"`
	Level int `json:"level"`
}

// BatchRecordsResponse batch 엔드포인트 응답
type BatchRecordsResponse struct {
	Results   []BatchResultItem `json:"results"`
	UserAfter BatchUserAfter    `json:"user_after"`
}
