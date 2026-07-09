package v1

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/witchs-lounge_backend/internal/delivery/http/handler"
)

// NewConsentLogRouter EULA/개인정보 동의 로그 적재 라우트.
//
// consent-log 합의(Q2): **무인증 개방** — record/event 와 달리 auth → ban → hmac 체인을
// 태우지 않는다. 동의는 로그인/세션 이전에 발생하기 때문이다. 세션(Bearer)이 있으면
// 핸들러가 옵션으로 user_id 를 채운다.
//
// 남용 방어: client_consent_id UNIQUE 멱등(usecase/DB) + IP 기준 rate-limit(아래).
func NewConsentLogRouter(router fiber.Router, consentLogHandler *handler.ConsentLogHandler) {
	con := router.Group("/consents")

	// IP 기준 rate-limit. 동의 UI 는 인트로 1회성이라 정상 트래픽은 낮음 —
	// 넉넉한 상한으로 오남용/봇 폭주만 차단(정상 재송신 batch 는 통과).
	rl := limiter.New(limiter.Config{
		Max:        60,
		Expiration: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
	})

	con.Post("/log", rl, consentLogHandler.CreateConsentLog)
	con.Post("/log/batch", rl, consentLogHandler.CreateConsentLogsBatch)
}
