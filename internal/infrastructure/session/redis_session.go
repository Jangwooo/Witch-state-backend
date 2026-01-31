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
	Get(ctx context.Context, sessionID string) (*entity.User, error)
	Delete(ctx context.Context, sessionID string) error
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
	sessionID := uuid.New().String()

	userData, err := json.Marshal(user)
	if err != nil {
		return "", fmt.Errorf("사용자 데이터 직렬화 중 오류: %w", err)
	}

	err = s.client.Set(ctx, "session:"+sessionID, userData, s.sessionTime).Err()
	if err != nil {
		return "", fmt.Errorf("Redis에 세션 저장 중 오류: %w", err)
	}

	return sessionID, nil
}

func (s *RedisSessionStore) Get(ctx context.Context, sessionID string) (*entity.User, error) {
	data, err := s.client.Get(ctx, "session:"+sessionID).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil // 세션 없음
		}
		return nil, fmt.Errorf("redis에서 세션 조회 중 오류: %w", err)
	}

	var user entity.User
	err = json.Unmarshal(data, &user)
	if err != nil {
		return nil, fmt.Errorf("사용자 데이터 역직렬화 중 오류: %w", err)
	}

	return &user, nil
}

func (s *RedisSessionStore) Delete(ctx context.Context, sessionID string) error {
	err := s.client.Del(ctx, "session:"+sessionID).Err()
	if err != nil {
		return fmt.Errorf("redis에서 세션 삭제 중 오류: %w", err)
	}
	return nil
}
