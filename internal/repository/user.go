package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/ent"
	"github.com/witchs-lounge_backend/ent/user"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/domain/repository"
)

type userRepository struct {
	client *ent.Client
}

func NewUserRepository(client *ent.Client) repository.UserRepository {
	return &userRepository{client: client}
}

func (r *userRepository) FindByPlatformUserID(ctx context.Context, platformType, platformUserID string) (*entity.User, error) {
	entUser, err := r.client.User.Query().
		Where(
			user.PlatformTypeEQ(user.PlatformType(platformType)),
			user.PlatformUserIDEQ(platformUserID),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	return entity.NewUser(entUser), nil
}

func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	entUser, err := r.client.User.Query().
		Where(user.ID(id)).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	return entity.NewUser(entUser), nil
}

// Create은 새로운 사용자를 생성하는 메소드입니다.
func (r *userRepository) Create(ctx context.Context, req *entity.CreateUserRequest) (*entity.User, error) {
	create := r.client.User.Create().
		SetPlatformType(user.PlatformType(req.PlatformType)).
		SetPlatformUserID(req.PlatformUserID).
		SetIsVerified(req.IsVerified).
		SetNickname(req.Nickname)

	// Optional 필드들 설정
	if req.PlatformEmail != "" {
		create = create.SetPlatformEmail(req.PlatformEmail)
	}
	if req.PlatformAvatarURL != "" {
		create = create.SetPlatformAvatarURL(req.PlatformAvatarURL)
	}
	if req.Language != "" {
		create = create.SetLanguage(req.Language)
	}
	if req.PlatformDisplayName != "" {
		create = create.SetPlatformDisplayName(req.PlatformDisplayName)
	}
	if req.PlatformData != nil {
		create = create.SetPlatformData(req.PlatformData)
	}

	entUser, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}

	return entity.NewUser(entUser), nil
}

// UpdatePlatformProfile updates platform-specific profile fields for a user.
func (r *userRepository) UpdatePlatformProfile(ctx context.Context, id uuid.UUID, displayName, avatarURL, nickname string, platformData map[string]interface{}) (*entity.User, error) {
	update := r.client.User.UpdateOneID(id)

	if displayName != "" {
		update = update.SetPlatformDisplayName(displayName)
	}
	if avatarURL != "" {
		update = update.SetPlatformAvatarURL(avatarURL)
	}
	if nickname != "" {
		update = update.SetNickname(nickname)
	}
	if platformData != nil {
		update = update.SetPlatformData(platformData)
	}

	entUser, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}

	return entity.NewUser(entUser), nil
}

// UpdateLastLogin은 사용자의 마지막 로그인 시간을 업데이트하는 메소드입니다.
func (r *userRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID, lastLoginTime time.Time) (*entity.User, error) {
	entUser, err := r.client.User.UpdateOneID(id).
		SetLastLoginAt(lastLoginTime).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return entity.NewUser(entUser), nil
}
