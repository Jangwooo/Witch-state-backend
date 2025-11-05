package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/ent"
	"github.com/witchs-lounge_backend/ent/record"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/domain/repository"
)

type recordRepository struct {
	client *ent.Client
}

func NewRecordRepository(client *ent.Client) repository.RecordRepository {
	return &recordRepository{client: client}
}

func (r *recordRepository) Create(ctx context.Context, userID uuid.UUID, req *entity.CreateRecordRequest) (*entity.Record, error) {
	rec, err := r.client.Record.Create().
		SetUserID(userID).
		SetMusicID(req.MusicID).
		SetStageID(req.StageID).
		SetCharacterID(req.CharacterID).
		SetScore(req.Score).
		SetPerfectCount(req.PerfectCount).
		SetGoodCount(req.GoodCount).
		SetBadCount(req.BadCount).
		SetMissCount(req.MissCount).
		SetMaxCombo(req.MaxCombo).
		SetAccuracy(req.Accuracy).
		SetRank(record.Rank(req.Rank)).
		SetIsFullCombo(req.IsFullCombo).
		SetIsPerfectPlay(req.IsPerfectPlay).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	// Optional 필드
	if req.PlayedAt != nil {
		_, _ = r.client.Record.UpdateOneID(rec.ID).SetPlayedAt(*req.PlayedAt).Save(ctx)
	}
	if req.PlayDuration != nil {
		_, _ = r.client.Record.UpdateOneID(rec.ID).SetPlayDuration(*req.PlayDuration).Save(ctx)
	}
	if req.Additional != nil {
		_, _ = r.client.Record.UpdateOneID(rec.ID).SetAdditionalInfo(req.Additional).Save(ctx)
	}

	return entity.NewRecord(rec), nil
}

func (r *recordRepository) ListByUserAndMusicStage(ctx context.Context, userID, musicID, stageID uuid.UUID) ([]*entity.Record, error) {
	recs, err := r.client.Record.Query().
		Where(
			record.UserIDEQ(userID),
			record.MusicIDEQ(musicID),
			record.StageIDEQ(stageID),
			record.IsValidEQ(true),
		).
		Order(ent.Desc(record.FieldPlayedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]*entity.Record, 0, len(recs))
	for _, v := range recs {
		res = append(res, entity.NewRecord(v))
	}
	return res, nil
}

func (r *recordRepository) BestByUserAndMusicStage(ctx context.Context, userID, musicID, stageID uuid.UUID) (*entity.Record, error) {
	rec, err := r.client.Record.Query().
		Where(
			record.UserIDEQ(userID),
			record.MusicIDEQ(musicID),
			record.StageIDEQ(stageID),
			record.IsValidEQ(true),
		).
		Order(
			ent.Desc(record.FieldScore),
			ent.Desc(record.FieldAccuracy),
			ent.Desc(record.FieldMaxCombo),
		).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return entity.NewRecord(rec), nil
}
