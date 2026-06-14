package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/witchs-lounge_backend/internal/delivery/http/handler"
	"github.com/witchs-lounge_backend/internal/delivery/http/middleware"
	"github.com/witchs-lounge_backend/internal/infrastructure/session"
)

func NewUserRouter(group fiber.Router, userHandler *handler.UserHandler, sessionStore session.SessionStore, banMW fiber.Handler) {
	user := group.Group("/api/v1/user")

	// 인증 → ban 체크
	user.Get("/me", middleware.AuthMiddleware(sessionStore), banMW, userHandler.GetMe)
}
