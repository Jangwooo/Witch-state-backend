package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/witchs-lounge_backend/internal/delivery/http/handler"
	"github.com/witchs-lounge_backend/internal/delivery/http/middleware"
	"github.com/witchs-lounge_backend/internal/infrastructure/session"
)

func NewRecordRouter(router fiber.Router, recordHandler *handler.RecordHandler, sessionStore session.SessionStore) {
	rec := router.Group("/api/v1/records")

	// 인증 필요
	rec.Post("/", middleware.AuthMiddleware(sessionStore), recordHandler.CreateRecord)
	rec.Get("/", middleware.AuthMiddleware(sessionStore), recordHandler.ListRecords)
	rec.Get("/best", middleware.AuthMiddleware(sessionStore), recordHandler.BestRecord)
}
