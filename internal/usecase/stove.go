package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/ent"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/domain/repository"
	"github.com/witchs-lounge_backend/internal/infrastructure/session"
)

// StoveUseCase Stove 인증 전용 UseCase
type StoveUseCase interface {
	SignInWithStove(ctx context.Context, info struct {
		ID          string
		Email       string
		AvatarUrl   string
		DisplayName string
	}) (*entity.SessionResponse, error)
	FindByID(ctx context.Context, id uuid.UUID) (*entity.UserResponse, error)
}

type stoveUseCase struct {
	userRepo     repository.UserRepository
	sessionStore session.SessionStore
}

// NewStoveUseCase Stove UseCase 생성자
func NewStoveUseCase(userRepo repository.UserRepository, sessionStore session.SessionStore) StoveUseCase {
	return &stoveUseCase{
		userRepo:     userRepo,
		sessionStore: sessionStore,
	}
}

// SignInWithStove 유저 검증 및 세션 생성
func (u *stoveUseCase) SignInWithStove(ctx context.Context, info struct {
	ID          string
	Email       string
	AvatarUrl   string
	DisplayName string
}) (*entity.SessionResponse, error) {
	user, err := u.userRepo.FindByPlatformUserID(ctx, "stove", info.ID)
	if err != nil {
		if ent.IsNotFound(err) {
			user, err = u.userRepo.Create(ctx, &entity.CreateUserRequest{
				PlatformType:        "stove",
				PlatformUserID:      info.ID,
				PlatformEmail:       info.Email,
				PlatformAvatarURL:   info.AvatarUrl,
				PlatformDisplayName: info.DisplayName,
				IsVerified:          true,
				Nickname:            info.DisplayName,
			})

			if err != nil {
				return nil, err
			}
		}
	}

	sid, err := u.sessionStore.Create(ctx, user)
	if err != nil {
		return nil, err
	}
	return user.ToSessionResponse(sid), nil
}

// FindByID ID로 사용자 조회
func (u *stoveUseCase) FindByID(ctx context.Context, id uuid.UUID) (*entity.UserResponse, error) {
	user, err := u.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return user.ToResponse(), nil
}
