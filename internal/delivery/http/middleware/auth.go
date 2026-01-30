package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/infrastructure/session"
)

// AuthMiddleware는 사용자 인증을 처리하는 미들웨어입니다.
func AuthMiddleware(sessionStore session.SessionStore) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Authorization 헤더에서 토큰 가져오기
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(entity.ErrorResponse{
				Message: "인증 토큰이 필요합니다",
				Error:   "missing_authorization_header",
			})
		}

		// "Bearer " 접두사 확인 및 제거
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			return c.Status(fiber.StatusUnauthorized).JSON(entity.ErrorResponse{
				Message: "유효하지 않은 인증 토큰 형식입니다",
				Error:   "invalid_authorization_header",
			})
		}
		sessionID := strings.TrimPrefix(authHeader, bearerPrefix)

		// 2. 세션 ID로 사용자 정보 조회
		user, err := sessionStore.Get(c.Context(), sessionID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(entity.ErrorResponse{
				Message: "세션 정보 조회 중 오류가 발생했습니다",
				Error:   err.Error(),
			})
		}

		if user == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(entity.ErrorResponse{
				Message: "유효하지 않은 세션입니다",
				Error:   "invalid_session",
			})
		}

		// 3. 사용자 정보를 컨텍스트에 저장
		c.Locals("user", user)
		c.Locals("sessionID", sessionID)

		// 다음 핸들러로 진행
		return c.Next()
	}
}
