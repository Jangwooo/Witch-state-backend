package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/witchs-lounge_backend/internal/delivery/http/handler"
	"github.com/witchs-lounge_backend/internal/delivery/http/middleware"
	"github.com/witchs-lounge_backend/internal/infrastructure/session"
)

func NewUserRouter(group fiber.Router, userHandler *handler.UserHandler, sessionStore session.SessionStore) {
	user := group.Group("/api/v1/user")

	// 인증이 필요한 엔드포인트
	user.Get("/me", middleware.AuthMiddleware(sessionStore), userHandler.GetMe)
}
