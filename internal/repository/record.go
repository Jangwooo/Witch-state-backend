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
		SetRank(record.Rank(entity.NormalizeRankForStorage(req.Rank))).
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

	// 전시(EXHIBITION) 기록은 is_valid=false 로 저장해 모든 집계(List/Best 등)에서 자동 제외한다
	// (exhibition-logging Q2). 마커는 additional_info.is_exhibition==true.
	// 비전시 기록은 세팅하지 않아 스키마 기본값(true)을 그대로 사용 — 기존 동작 불변.
	if isExhibitionAdditional(req.Additional) {
		create.SetIsValid(false)
	}

	rec, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}

	return entity.NewRecord(rec), nil
}

// isExhibitionAdditional 은 additional_info.is_exhibition==true 여부를 판정한다
// (exhibition-logging Q1/Q2). usecase.isExhibitionRequest 와 동일 규칙 — repo 가 usecase 를
// import 하면 순환이 되므로 map 기준 로직만 로컬 복제.
func isExhibitionAdditional(additional map[string]interface{}) bool {
	if additional == nil {
		return false
	}
	v, ok := additional["is_exhibition"]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
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
