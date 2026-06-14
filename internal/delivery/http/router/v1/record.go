package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/witchs-lounge_backend/internal/delivery/http/handler"
	"github.com/witchs-lounge_backend/internal/delivery/http/middleware"
	"github.com/witchs-lounge_backend/internal/infrastructure/session"
)

func NewRecordRouter(router fiber.Router, recordHandler *handler.RecordHandler, sessionStore session.SessionStore, hmacMW fiber.Handler) {
	rec := router.Group("/records")

	auth := middleware.AuthMiddleware(sessionStore)

	// 인증 + HMAC 검증 (HMAC_MODE=off 면 no-op)
	rec.Post("/", auth, hmacMW, recordHandler.CreateRecord)
	rec.Get("/", auth, hmacMW, recordHandler.ListRecords)
	rec.Get("/best", auth, hmacMW, recordHandler.BestRecord)
}
