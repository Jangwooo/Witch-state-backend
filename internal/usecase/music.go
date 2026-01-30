package usecase

import (
	"context"
	"time"

	"github.com/witchs-lounge_backend/internal/dto"
	"github.com/witchs-lounge_backend/internal/repository"
)

type MusicUseCase interface {
	GetActiveMusics(ctx context.Context) ([]*dto.MusicResponse, error)
}

type musicUseCase struct {
	musicRepo repository.MusicRepository
}

func NewMusicUseCase(musicRepo repository.MusicRepository) MusicUseCase {
	return &musicUseCase{musicRepo: musicRepo}
}

func (u *musicUseCase) GetActiveMusics(ctx context.Context) ([]*dto.MusicResponse, error) {
	musics, err := u.musicRepo.GetActiveMusics(ctx)
	if err != nil {
		return nil, err
	}

	var musicResponses []*dto.MusicResponse
	for _, m := range musics {
		var releaseDateStr *string
		if m.ReleaseDate != nil {
			s := m.ReleaseDate.Format(time.RFC3339)
			releaseDateStr = &s
		}
		musicResponses = append(musicResponses, &dto.MusicResponse{
			ID:            m.ID,
			Name:          m.Name,
			Artist:        m.Artist,
			Composer:      m.Composer,
			Bpm:           m.Bpm,
			Genre:         m.Genre,
			Description:   m.Description,
			IsRecommended: m.IsRecommended,
			IsFree:        m.IsFree,
			UnlockLevel:   m.UnlockLevel,
			ReleaseDate:   releaseDateStr,
		})
	}
	return musicResponses, nil
}
