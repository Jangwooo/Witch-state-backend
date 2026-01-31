package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/ent"
)

// User represents the domain entity for a user
type User struct {
	*ent.User // ent 엔티티 임베드
}

// PlatformUserInfo 플랫폼에서 가져온 사용자 정보
type PlatformUserInfo struct {
	PlatformType   string                 `json:"platform_type"`
	PlatformUserID string                 `json:"platform_user_id"`
	Email          string                 `json:"email"`
	AvatarURL      string                 `json:"avatar_url"`
	DisplayName    string                 `json:"display_name"`
	Language       string                 `json:"language"`
	PlatformData   map[string]interface{} `json:"platform_data"`
	IsVerified     bool                   `json:"is_verified"`
}

// CreateUserRequest 새 사용자 생성 요청 데이터
type CreateUserRequest struct {
	PlatformType        string                 `json:"platform_type"`
	PlatformUserID      string                 `json:"platform_user_id"`
	PlatformEmail       string                 `json:"platform_email"`
	PlatformAvatarURL   string                 `json:"platform_avatar_url"`
	PlatformDisplayName string                 `json:"platform_display_name"`
	Language            string                 `json:"language"`
	PlatformData        map[string]interface{} `json:"platform_data"`
	IsVerified          bool                   `json:"is_verified"`
	Nickname            string                 `json:"nickname"`
}

// UserResponse represents the response for user data
type UserResponse struct {
	ID                  uuid.UUID `json:"id"`
	PlatformType        string    `json:"platform_type"`
	PlatformUserID      string    `json:"platform_user_id"`
	PlatformEmail       string    `json:"platform_email"`
	PlatformAvatarURL   string    `json:"platform_avatar_url"`
	PlatformDisplayName string    `json:"platform_display_name"`
	Language            string    `json:"language"`
	IsVerified          bool      `json:"is_verified"`
	Nickname            string    `json:"nickname"`
	DisplayName         string    `json:"display_name"`
	Level               int       `json:"level"`
	Exp                 int       `json:"exp"`
	Coin                int       `json:"coin"`
	Gem                 int       `json:"gem"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// SessionResponse 세션 정보와 유저 정보를 담는 응답 구조체
type SessionResponse struct {
	SessionID string       `json:"session_id"`
	User      UserResponse `json:"user"`
}

// StoveSignInRequest Stove 로그인 요청 구조체
type StoveSignInRequest struct {
	AccessToken string `validate:"required" json:"access_token"`
}

// SteamSignInRequest Steam 로그인 요청 구조체
type SteamSignInRequest struct {
	Id     string `validate:"required" json:"id"`
	Ticket string `validate:"required" json:"ticket"`
}

// NewUser creates a new User instance
func NewUser(entUser *ent.User) *User {
	return &User{
		User: entUser,
	}
}

// ToResponse converts User to UserResponse
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:                  u.ID,
		PlatformType:        string(u.PlatformType),
		PlatformUserID:      u.PlatformUserID,
		PlatformEmail:       u.PlatformEmail,
		PlatformAvatarURL:   u.PlatformAvatarURL,
		PlatformDisplayName: u.PlatformDisplayName,
		Language:            u.Language,
		IsVerified:          u.IsVerified,
		Nickname:            u.Nickname,
		DisplayName:         u.DisplayName,
		Level:               u.Level,
		Exp:                 u.Exp,
		Coin:                u.Coin,
		Gem:                 u.Gem,
		CreatedAt:           u.CreatedAt,
		UpdatedAt:           u.UpdatedAt,
	}
}

// ToSessionResponse converts User and sessionID to SessionResponse
func (u *User) ToSessionResponse(sessionID string) *SessionResponse {
	return &SessionResponse{
		SessionID: sessionID,
		User:      *u.ToResponse(),
	}
}

// FromEntUser converts an ent.User to our domain User
func FromEntUser(entUser *ent.User) *User {
	return &User{
		User: entUser,
	}
}
