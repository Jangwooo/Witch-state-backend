package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/witchs-lounge_backend/internal/domain/entity"
)

type SessionStore interface {
	Create(ctx context.Context, user *entity.User) (string, error)
	CreateWithSecret(ctx context.Context, user *entity.User, hmacSecret string) (string, error)
	Get(ctx context.Context, sessionID string) (*entity.User, error)
	GetWithSecret(ctx context.Context, sessionID string) (*entity.User, string, error)
	Delete(ctx context.Context, sessionID string) error
}

// sessionPayload 는 신규 세션 페이로드 포맷입니다.
// 구버전(= entity.User 단독 직렬화) 호환을 위해 Get/GetWithSecret 에서
// 두 가지 포맷 모두 디코딩을 시도합니다.
type sessionPayload struct {
	User       *entity.User `json:"user"`
	HmacSecret string       `json:"hmac_secret,omitempty"`
}

type RedisCommand interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

// RedisSessionStore는 Redis를 사용한 세션 스토어입니다.
type RedisSessionStore struct {
	client      RedisCommand
	sessionTime time.Duration
}

// NewRedisSessionStore는 단일 Redis 클라이언트를 사용하는 세션 스토어를 생성합니다.
func NewRedisSessionStore(client *redis.Client, sessionTime time.Duration) *RedisSessionStore {
	if sessionTime == 0 {
		sessionTime = 24 * time.Hour // 기본 세션 시간 - 24시간
	}
	return &RedisSessionStore{
		client:      client,
		sessionTime: sessionTime,
	}
}

// NewRedisClusterSessionStore는 Redis 클러스터 클라이언트를 사용하는 세션 스토어를 생성합니다.
func NewRedisClusterSessionStore(clusterClient *redis.ClusterClient, sessionTime time.Duration) *RedisSessionStore {
	if sessionTime == 0 {
		sessionTime = 24 * time.Hour // 기본 세션 시간 - 24시간
	}
	return &RedisSessionStore{
		client:      clusterClient,
		sessionTime: sessionTime,
	}
}

func (s *RedisSessionStore) Create(ctx context.Context, user *entity.User) (string, error) {
	return s.CreateWithSecret(ctx, user, "")
}

func (s *RedisSessionStore) CreateWithSecret(ctx context.Context, user *entity.User, hmacSecret string) (string, error) {
	sessionID := uuid.New().String()

	payload := sessionPayload{
		User:       user,
		HmacSecret: hmacSecret,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("세션 페이로드 직렬화 중 오류: %w", err)
	}

	err = s.client.Set(ctx, "session:"+sessionID, data, s.sessionTime).Err()
	if err != nil {
		return "", fmt.Errorf("Redis에 세션 저장 중 오류: %w", err)
	}

	return sessionID, nil
}

func (s *RedisSessionStore) Get(ctx context.Context, sessionID string) (*entity.User, error) {
	user, _, err := s.GetWithSecret(ctx, sessionID)
	return user, err
}

// GetWithSecret 은 신규 페이로드 포맷({user, hmac_secret})을 우선 디코딩하고,
// 실패 시 구버전(entity.User 단독) 포맷으로 폴백합니다.
// 구버전 세션은 hmacSecret 이 빈 문자열로 반환됩니다 — 검증 미들웨어가 grace period 로 스킵합니다.
func (s *RedisSessionStore) GetWithSecret(ctx context.Context, sessionID string) (*entity.User, string, error) {
	data, err := s.client.Get(ctx, "session:"+sessionID).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, "", nil // 세션 없음
		}
		return nil, "", fmt.Errorf("redis에서 세션 조회 중 오류: %w", err)
	}

	// 신규 포맷 우선 시도. User 필드가 존재해야 신규 포맷으로 인정.
	var payload sessionPayload
	if err := json.Unmarshal(data, &payload); err == nil && payload.User != nil {
		return payload.User, payload.HmacSecret, nil
	}

	// 구버전 폴백: entity.User 단독 직렬화
	var legacy entity.User
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, "", fmt.Errorf("사용자 데이터 역직렬화 중 오류: %w", err)
	}
	return &legacy, "", nil
}

func (s *RedisSessionStore) Delete(ctx context.Context, sessionID string) error {
	err := s.client.Del(ctx, "session:"+sessionID).Err()
	if err != nil {
		return fmt.Errorf("redis에서 세션 삭제 중 오류: %w", err)
	}
	return nil
}
