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
	return r.create(ctx, userID, "", req)
}

func (r *recordRepository) CreateWithClientRecordID(ctx context.Context, userID uuid.UUID, clientRecordID string, req *entity.CreateRecordRequest) (*entity.Record, error) {
	return r.create(ctx, userID, clientRecordID, req)
}

func (r *recordRepository) create(ctx context.Context, userID uuid.UUID, clientRecordID string, req *entity.CreateRecordRequest) (*entity.Record, error) {
	create := r.client.Record.Create().
		SetUserID(userID).
		SetMusicID(req.MusicID).
		SetStageID(req.StageID).
		SetScore(req.Score).
		SetPerfectCount(req.PerfectCount).
		SetGoodCount(req.GoodCount).
		SetBadCount(req.BadCount).
		SetMissCount(req.MissCount).
		SetMaxCombo(req.MaxCombo).
		SetAccuracy(req.Accuracy).
		SetRank(record.Rank(req.Rank)).
		SetIsFullCombo(req.IsFullCombo).
		SetIsPerfectPlay(req.IsPerfectPlay)

	if clientRecordID != "" {
		create.SetClientRecordID(clientRecordID)
	}

	// GameStatus 처리 (기본값: completed)
	if req.GameStatus != "" {
		create.SetGameStatus(record.GameStatus(req.GameStatus))
	}

	if req.Additional != nil {
		create.SetAdditionalInfo(req.Additional)
	} else {
		create.SetAdditionalInfo(map[string]interface{}{})
	}

	rec, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}

	return entity.NewRecord(rec), nil
}

func (r *recordRepository) FindByClientRecordID(ctx context.Context, clientRecordID string) (*entity.Record, error) {
	rec, err := r.client.Record.Query().
		Where(record.ClientRecordIDEQ(clientRecordID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return entity.NewRecord(rec), nil
}

func (r *recordRepository) ListByUserAndMusicStage(ctx context.Context, userID uuid.UUID, musicID, stageID string) ([]*entity.Record, error) {
	recs, err := r.client.Record.Query().
		Where(
			record.UserIDEQ(userID),
			record.MusicIDEQ(musicID),
			record.StageIDEQ(stageID),
			record.IsValidEQ(true),
		).
		Order(ent.Desc(record.FieldCreatedAt)).
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

func (r *recordRepository) BestByUserAndMusicStage(ctx context.Context, userID uuid.UUID, musicID, stageID string) (*entity.Record, error) {
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
