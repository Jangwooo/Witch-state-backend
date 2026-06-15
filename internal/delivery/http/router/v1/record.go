package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/witchs-lounge_backend/internal/delivery/http/handler"
	"github.com/witchs-lounge_backend/internal/delivery/http/middleware"
	"github.com/witchs-lounge_backend/internal/infrastructure/session"
)

func NewRecordRouter(router fiber.Router, recordHandler *handler.RecordHandler, sessionStore session.SessionStore, banMW, hmacMW fiber.Handler) {
	rec := router.Group("/records")

	auth := middleware.AuthMiddleware(sessionStore)

	// 인증 → ban 체크(403 account_banned) → HMAC 검증(HMAC_MODE=off 면 no-op)
	rec.Post("/", auth, banMW, hmacMW, recordHandler.CreateRecord)
	rec.Post("/batch", auth, banMW, hmacMW, recordHandler.CreateRecordsBatch)
	rec.Get("/", auth, banMW, hmacMW, recordHandler.ListRecords)
	rec.Get("/best", auth, banMW, hmacMW, recordHandler.BestRecord)
}
