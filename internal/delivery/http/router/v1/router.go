package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/witchs-lounge_backend/internal/delivery/http/handler"
	"github.com/witchs-lounge_backend/internal/infrastructure/session"
)

// RouterConfig 라우터 설정에 필요한 의존성
type RouterConfig struct {
	SessionStore session.SessionStore

	StoveHandler  *handler.StoveHandler
	UserHandler   *handler.UserHandler
	RecordHandler *handler.RecordHandler
}

// SetupRoutes 모든 라우터를 마운트
func SetupRoutes(app *fiber.App, config *RouterConfig) {
	// V1 API 라우터 등록
	NewStoveRouter(app, config.StoveHandler)
	NewUserRouter(app, config.UserHandler, config.SessionStore)
	NewRecordRouter(app, config.RecordHandler, config.SessionStore)

	// 추가 라우터는 여기에 등록
	// 인증 불필요: NewPublicRouter(app, handler)
	// 인증 필요: NewProtectedRouter(app, handler, config.SessionStore)
}
