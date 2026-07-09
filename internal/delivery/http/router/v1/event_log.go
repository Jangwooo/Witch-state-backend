package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/witchs-lounge_backend/internal/delivery/http/handler"
	"github.com/witchs-lounge_backend/internal/delivery/http/middleware"
	"github.com/witchs-lounge_backend/internal/infrastructure/session"
)

// NewEventLogRouter Event 상태변경 로그 적재 라우트.
// record 와 동일한 auth → ban → hmac 체인을 쓰되, exhibitionGate 는 미적용한다
// (event-server-log 합의: 전시 빌드는 세션이 없어 로그를 보내지 않음 — 클라 EventLogReporter 도 전시 빌드에서 비활성).
func NewEventLogRouter(router fiber.Router, eventLogHandler *handler.EventLogHandler, sessionStore session.SessionStore, banMW, hmacMW fiber.Handler) {
	ev := router.Group("/events")

	auth := middleware.AuthMiddleware(sessionStore)

	// 인증 → ban 체크(403 account_banned) → HMAC 검증(HMAC_MODE=off 면 no-op).
	ev.Post("/log", auth, banMW, hmacMW, eventLogHandler.CreateEventLog)
	ev.Post("/log/batch", auth, banMW, hmacMW, eventLogHandler.CreateEventLogsBatch)
}
