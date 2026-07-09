package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/ent"
	"github.com/witchs-lounge_backend/ent/eventlog"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/domain/repository"
)

type eventLogRepository struct {
	client *ent.Client
}

func NewEventLogRepository(client *ent.Client) repository.EventLogRepository {
	return &eventLogRepository{client: client}
}

func (r *eventLogRepository) CreateWithClientLogID(ctx context.Context, userID uuid.UUID, clientLogID string, req *entity.CreateEventLogRequest) (*entity.EventLog, error) {
	create := r.client.EventLog.Create().
		SetUserID(userID).
		SetEventKey(req.EventKey).
		SetStateBefore(req.StateBefore).
		SetStateAfter(req.StateAfter).
		SetChangedAt(req.ChangedAtTime())

	if clientLogID != "" {
		create.SetClientLogID(clientLogID)
	}

	logEnt, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}
	return entity.NewEventLog(logEnt), nil
}

func (r *eventLogRepository) FindByClientLogID(ctx context.Context, clientLogID string) (*entity.EventLog, error) {
	logEnt, err := r.client.EventLog.Query().
		Where(eventlog.ClientLogIDEQ(clientLogID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return entity.NewEventLog(logEnt), nil
}
