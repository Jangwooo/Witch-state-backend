package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/witchs-lounge_backend/internal/delivery/http/handler"
	"github.com/witchs-lounge_backend/internal/delivery/http/middleware"
	"github.com/witchs-lounge_backend/internal/infrastructure/hmacauth"
	"github.com/witchs-lounge_backend/internal/infrastructure/logging"
	"github.com/witchs-lounge_backend/internal/infrastructure/session"
)

// RouterConfig 라우터 설정에 필요한 의존성
type RouterConfig struct {
	SessionStore session.SessionStore
	RedisClient  *redis.Client
	ErrorLogger  *logging.ErrorLogger
	HMACConfig   *hmacauth.Config

	StoveHandler  *handler.StoveHandler
	SteamHandler  *handler.SteamHandler
	UserHandler   *handler.UserHandler
	RecordHandler *handler.RecordHandler
	MusicHandler  *handler.MusicHandler
	StageHandler  *handler.StageHandler
}

// SetupRoutes 모든 라우터를 마운트
func SetupRoutes(app *fiber.App, config *RouterConfig) {
	// V1 API 라우터 등록

	v1 := app.Group("/api/v1")

	// HMAC 검증 미들웨어를 한 번 생성해 인증 필요 엔드포인트에 재사용.
	hmacMW := middleware.HMACMiddleware(middleware.HMACVerifierConfig{
		Config:      config.HMACConfig,
		NonceStore:  &middleware.RedisNonceStore{Client: config.RedisClient},
		ErrorLogger: config.ErrorLogger,
	})

	NewStoveRouter(v1, config.StoveHandler)
	NewSteamRouter(v1, config.SteamHandler)
	NewUserRouter(v1, config.UserHandler, config.SessionStore)
	NewRecordRouter(v1, config.RecordHandler, config.SessionStore, hmacMW)
	MusicRouter(v1, config.MusicHandler)
	StageRouter(v1, config.StageHandler)

	// 추가 라우터는 여기에 등록
	// 인증 불필요: NewPublicRouter(app, handler)
	// 인증 필요: NewProtectedRouter(app, handler, config.SessionStore)
}
