package repository

import (
	"context"

	"github.com/witchs-lounge_backend/ent"
	"github.com/witchs-lounge_backend/ent/music"
)

type MusicRepository interface {
	GetActiveMusics(ctx context.Context) ([]*ent.Music, error)
	GetMusicByID(ctx context.Context, musicID string) (*ent.Music, error)
}

type musicRepository struct {
	client *ent.Client
}

func NewMusicRepository(dbClient *ent.Client) MusicRepository {
	return &musicRepository{client: dbClient}
}

func (r *musicRepository) GetActiveMusics(ctx context.Context) ([]*ent.Music, error) {
	return r.client.Music.Query().
		Where(music.IsActive(true)).
		All(ctx)
}

func (r *musicRepository) GetMusicByID(ctx context.Context, musicID string) (*ent.Music, error) {
	return r.client.Music.Query().
		Where(music.ID(musicID)).
		Only(ctx)
}
