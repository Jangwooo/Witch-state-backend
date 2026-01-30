package repository

import (
	"context"

	"github.com/witchs-lounge_backend/ent"
	"github.com/witchs-lounge_backend/ent/stage"
)

type StageRepository interface {
	GetStagesByMusicID(ctx context.Context, musicID string) ([]*ent.Stage, error)
}

type stageRepository struct {
	client *ent.Client
}

func NewStageRepository(client *ent.Client) StageRepository {
	return &stageRepository{client: client}
}

func (r *stageRepository) GetStagesByMusicID(ctx context.Context, musicID string) ([]*ent.Stage, error) {
	return r.client.Stage.Query().
		Where(stage.MusicID(musicID), stage.IsActive(true)).
		All(ctx)
}
