package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/witchs-lounge_backend/internal/delivery/http/handler"
	"github.com/witchs-lounge_backend/internal/delivery/http/middleware"
	"github.com/witchs-lounge_backend/internal/infrastructure/session"
)

func NewRecordRouter(router fiber.Router, recordHandler *handler.RecordHandler, sessionStore session.SessionStore, exhibitionGate, banMW, hmacMW fiber.Handler) {
	rec := router.Group("/records")

	auth := middleware.AuthMiddleware(sessionStore)

	// 전시 게이트 → 인증 → ban 체크(403 account_banned) → HMAC 검증(HMAC_MODE=off 면 no-op)
	// 전시 게이트가 X-Exhibition-Key 를 확인해 전시 요청이면 후속 미들웨어(auth/ban/hmac)가 각자 우회.
	// 전시는 단건 POST 만 사용하지만(batch 미사용), 게이트는 no-op 성격이라 모든 record 라우트에 동일 적용해도 안전.
	rec.Post("/", exhibitionGate, auth, banMW, hmacMW, recordHandler.CreateRecord)
	rec.Post("/batch", exhibitionGate, auth, banMW, hmacMW, recordHandler.CreateRecordsBatch)
	rec.Get("/", exhibitionGate, auth, banMW, hmacMW, recordHandler.ListRecords)
	rec.Get("/best", exhibitionGate, auth, banMW, hmacMW, recordHandler.BestRecord)
}
