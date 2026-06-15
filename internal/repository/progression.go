package repository

import (
	"context"

	"github.com/witchs-lounge_backend/ent"
	"github.com/witchs-lounge_backend/ent/tlevel"
	"github.com/witchs-lounge_backend/ent/trank"
)

type ProgressionRepository interface {
	GetRankCoefficient(ctx context.Context, rank string) (float32, error)
	GetHighestLevelForExp(ctx context.Context, exp int) (int, error)
}

type progressionRepository struct {
	client *ent.Client
}

func NewProgressionRepository(client *ent.Client) ProgressionRepository {
	return &progressionRepository{client: client}
}

func (r *progressionRepository) GetRankCoefficient(ctx context.Context, rank string) (float32, error) {
	row, err := r.client.TRank.Query().Where(trank.ID(rank)).Only(ctx)
	if err != nil {
		return 0, err
	}
	return row.Coefficient, nil
}

// GetHighestLevelForExp returns the highest level whose require_exp <= exp.
func (r *progressionRepository) GetHighestLevelForExp(ctx context.Context, exp int) (int, error) {
	row, err := r.client.TLevel.Query().
		Where(tlevel.RequireExpLTE(exp)).
		Order(ent.Desc(tlevel.FieldID)).
		First(ctx)
	if err != nil {
		return 0, err
	}
	return row.ID, nil
}
