package bootstrap

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/witchs-lounge_backend/internal/infrastructure/database"
	redisClient "github.com/witchs-lounge_backend/internal/infrastructure/redis"
	"github.com/witchs-lounge_backend/internal/infrastructure/session"
)

// SetupRedis Redis 연결 초기화 및 세션 스토어 생성
func SetupRedis(mode string) (*redis.Client, session.SessionStore, error) {
	dbConfig, err := database.LoadConfig(&mode)
	if err != nil {
		return nil, nil, err
	}

	log.Printf("Redis 연결 중: %s", dbConfig.RedisHost)

	// Redis 클라이언트 생성
	client := redisClient.NewRedisClient(dbConfig.RedisHost, dbConfig.RedisPassword, dbConfig.RedisDB)

	// Redis 연결 테스트
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = client.Ping(ctx).Result()
	if err != nil {
		return client, nil, err
	}

	log.Printf("Redis 연결 성공!")

	// 세션 스토어 초기화
	sessionStore := session.NewRedisSessionStore(client, 24*time.Hour)

	return client, sessionStore, nil
}
