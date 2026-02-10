package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/internal/domain/entity"
)

type UserRepository interface {
	FindByPlatformUserID(ctx context.Context, platformType, platformUserID string) (*entity.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	Create(ctx context.Context, user *entity.CreateUserRequest) (*entity.User, error)
	UpdatePlatformProfile(ctx context.Context, id uuid.UUID, displayName, avatarURL, nickname string, platformData map[string]interface{}) (*entity.User, error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID, lastLoginTime time.Time) (*entity.User, error)
}
