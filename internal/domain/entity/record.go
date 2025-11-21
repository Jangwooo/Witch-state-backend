package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/ent"
)

// Record 도메인 엔티티 래퍼
type Record struct {
	*ent.Record
}

type CreateRecordRequest struct {
	MusicID       string                 `json:"music_id" validate:"required"`
	StageID       string                 `json:"stage_id" validate:"required"`
	CharacterID   uuid.UUID              `json:"character_id" validate:"required"`
	Score         int                    `json:"score" validate:"required,gte=0"`
	PerfectCount  int                    `json:"perfect_count"`
	GoodCount     int                    `json:"good_count"`
	BadCount      int                    `json:"bad_count"`
	MissCount     int                    `json:"miss_count"`
	MaxCombo      int                    `json:"max_combo"`
	Accuracy      float64                `json:"accuracy"`
	Rank          string                 `json:"rank"`
	IsFullCombo   bool                   `json:"is_full_combo"`
	IsPerfectPlay bool                   `json:"is_perfect_play"`
	PlayDuration  *int                   `json:"play_duration,omitempty"`
	Additional    map[string]interface{} `json:"additional_info,omitempty"`
}

type RecordResponse struct {
	ID            uuid.UUID              `json:"id"`
	UserID        uuid.UUID              `json:"user_id"`
	MusicID       string                 `json:"music_id"`
	StageID       string                 `json:"stage_id"`
	CharacterID   uuid.UUID              `json:"character_id"`
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
	PlayedAt      time.Time              `json:"played_at"`
	PlayDuration  int                    `json:"play_duration"`
	Additional    map[string]interface{} `json:"additional_info,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

type CreateRecordResponse struct {
	Record     RecordResponse   `json:"record"`
	AllRecords []RecordResponse `json:"all_records"`
	BestRecord *RecordResponse  `json:"best_record"`
}

func NewRecord(entRec *ent.Record) *Record {
	return &Record{Record: entRec}
}

func (r *Record) ToResponse() RecordResponse {
	return RecordResponse{
		ID:            r.ID,
		UserID:        r.UserID,
		MusicID:       r.MusicID,
		StageID:       r.StageID,
		CharacterID:   r.CharacterID,
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
		PlayDuration:  r.PlayDuration,
		Additional:    r.AdditionalInfo,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}
