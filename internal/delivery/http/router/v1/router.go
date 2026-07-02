package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/witchs-lounge_backend/internal/delivery/http/handler"
	"github.com/witchs-lounge_backend/internal/delivery/http/middleware"
	"github.com/witchs-lounge_backend/internal/domain/repository"
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
	UserRepo     repository.UserRepository
	BanCache     *middleware.BanCache

	// 전시(EXHIBITION) 게이트 설정. 미설정 시 게이트 비활성 (기존 동작 불변).
	ExhibitionGate middleware.ExhibitionGateConfig

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

	// Ban 체크 미들웨어. AuthMiddleware 직후, HMAC 미들웨어 직전에 둔다.
	banMW := middleware.BanCheckMiddleware(config.UserRepo, config.BanCache)

	NewStoveRouter(v1, config.StoveHandler)
	NewSteamRouter(v1, config.SteamHandler)
	NewUserRouter(v1, config.UserHandler, config.SessionStore, banMW)
	// 전시 게이트: record 라우트 최전단. 유효한 X-Exhibition-Key 면 세션/밴/HMAC 우회.
	// 미설정 시 no-op (모든 요청이 기존 auth 체인을 그대로 탐).
	exhibitionGate := middleware.ExhibitionGate(config.ExhibitionGate)

	NewRecordRouter(v1, config.RecordHandler, config.SessionStore, exhibitionGate, banMW, hmacMW)
	MusicRouter(v1, config.MusicHandler)
	StageRouter(v1, config.StageHandler)

	// 추가 라우터는 여기에 등록
	// 인증 불필요: NewPublicRouter(app, handler)
	// 인증 필요: NewProtectedRouter(app, handler, config.SessionStore)
}
