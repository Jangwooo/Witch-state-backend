package redis

import (
	"strings"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient 은 단일 Redis 서버에 연결하는 클라이언트를 생성합니다.
func NewRedisClient(addr, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:            addr,
		Password:        password,
		DB:              db,
		MaxRetries:      5,
		MinRetryBackoff: 300,
		MaxRetryBackoff: 500,
	})
}

// NewRedisClusterClient 는 Redis 클러스터에 연결하는 클라이언트를 생성합니다.
// 여러 주소를 쉼표로 구분하여 입력 가능합니다: "redis1:6379,redis2:6380,redis3:6381"
func NewRedisClusterClient(addrs, password string) *redis.ClusterClient {
	addrList := strings.Split(addrs, ",")
	return redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:          addrList,
		Password:       password,
		RouteByLatency: true,
		MaxRetries:     5,
		ReadOnly:       true,
	})
}
