package middleware

import (
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/infrastructure/logging"
)

func ErrorLoggingMiddleware(errorLogger *logging.ErrorLogger) fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				errorLogger.LogPanic(c, recovered, debug.Stack())
				err = c.Status(fiber.StatusInternalServerError).JSON(entity.ErrorResponse{
					Message: "서버 내부 오류",
					Error:   "internal server error",
				})
			}
		}()

		err = c.Next()
		if err == nil && c.Response().StatusCode() >= fiber.StatusInternalServerError {
			errorLogger.LogHTTPError(c, c.Response().StatusCode(), nil)
		}

		return err
	}
}
