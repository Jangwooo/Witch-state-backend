package usecase

import (
	"context"
	"errors"

	"github.com/witchs-lounge_backend/internal/dto"
	"github.com/witchs-lounge_backend/internal/repository"
)

type StageUseCase interface {
	GetStagesByMusicID(ctx context.Context, musicID string) ([]*dto.StageResponse, error)
}

type stageUseCase struct {
	stageRepo repository.StageRepository
	musicRepo repository.MusicRepository
}

func NewStageUseCase(stageRepo repository.StageRepository, musicRepo repository.MusicRepository) StageUseCase {
	return &stageUseCase{stageRepo: stageRepo, musicRepo: musicRepo}
}

func (u *stageUseCase) GetStagesByMusicID(ctx context.Context, musicID string) ([]*dto.StageResponse, error) {
	music, err := u.musicRepo.GetMusicByID(ctx, musicID)
	if err != nil {
		return nil, err
	}
	if !music.IsActive {
		return nil, errors.New("music is not active")
	}

	stages, err := u.stageRepo.GetStagesByMusicID(ctx, musicID)
	if err != nil {
		return nil, err
	}

	var stageResponses []*dto.StageResponse
	for _, s := range stages {
		stageResponses = append(stageResponses, &dto.StageResponse{
			ID:         s.ID,
			MusicID:    s.MusicID,
			LevelName:  s.LevelName,
			Difficulty: s.Difficulty,
			TotalNotes: s.TotalNotes,
			MaxCombo:   s.MaxCombo,
		})
	}

	return stageResponses, nil
}
