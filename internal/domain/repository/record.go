package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/internal/domain/entity"
)

type RecordRepository interface {
	Create(ctx context.Context, userID uuid.UUID, req *entity.CreateRecordRequest) (*entity.Record, error)
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

type SingleRecordResponse struct {
	Record RecordResponse `json:"record"`
}

type ListRecordResponse struct {
	Records []RecordResponse `json:"records"`
}

type BestRecordResponse struct {
	Record *RecordResponse `json:"record"`
}
