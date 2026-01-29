package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/domain/repository"
)

type RecordUseCase interface {
	Create(ctx context.Context, userID uuid.UUID, req *entity.CreateRecordRequest) (*repository.SingleRecordResponse, error)
	List(ctx context.Context, userID uuid.UUID, musicID, stageID string) (*repository.ListRecordResponse, error)
	Best(ctx context.Context, userID uuid.UUID, musicID, stageID string) (*repository.BestRecordResponse, error)
}

type recordUseCase struct {
	recordRepo repository.RecordRepository
}

func NewRecordUseCase(recordRepo repository.RecordRepository) RecordUseCase {
	return &recordUseCase{recordRepo: recordRepo}
}

func (u *recordUseCase) Create(ctx context.Context, userID uuid.UUID, req *entity.CreateRecordRequest) (*repository.SingleRecordResponse, error) {
	created, err := u.recordRepo.Create(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	rr := toRecordResponse(created)
	return &repository.SingleRecordResponse{Record: rr}, nil
}

func (u *recordUseCase) List(ctx context.Context, userID uuid.UUID, musicID, stageID string) (*repository.ListRecordResponse, error) {
	list, err := u.recordRepo.ListByUserAndMusicStage(ctx, userID, musicID, stageID)
	if err != nil {
		return nil, err
	}
	out := make([]repository.RecordResponse, 0, len(list))
	for _, v := range list {
		out = append(out, toRecordResponse(v))
	}
	return &repository.ListRecordResponse{Records: out}, nil
}

func (u *recordUseCase) Best(ctx context.Context, userID uuid.UUID, musicID, stageID string) (*repository.BestRecordResponse, error) {
	best, err := u.recordRepo.BestByUserAndMusicStage(ctx, userID, musicID, stageID)
	if err != nil {
		return &repository.BestRecordResponse{Record: nil}, nil
	}
	if best == nil {
		return &repository.BestRecordResponse{Record: nil}, nil
	}
	br := toRecordResponse(best)
	return &repository.BestRecordResponse{Record: &br}, nil
}

// toRecordResponse: entity.Record -> repository.RecordResponse
func toRecordResponse(r *entity.Record) repository.RecordResponse {
	return repository.RecordResponse{
		ID:            r.ID,
		UserID:        r.UserID,
		MusicID:       r.MusicID,
		StageID:       r.StageID,
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
		Additional:    r.AdditionalInfo,
		CreatedAt:     r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     r.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
