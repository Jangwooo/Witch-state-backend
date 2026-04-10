package main

import (
	"errors"
	"flag"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/swagger"
	"github.com/redis/go-redis/v9"
	_ "github.com/witchs-lounge_backend/docs"
	"github.com/witchs-lounge_backend/ent"
	"github.com/witchs-lounge_backend/internal/delivery/http/middleware"
	v1 "github.com/witchs-lounge_backend/internal/delivery/http/router/v1"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/infrastructure/bootstrap"
	appLogging "github.com/witchs-lounge_backend/internal/infrastructure/logging"
)

// @title           Witch's Lounge API
// @version         1.0
// @description     Witch's Lounge Backend API Server
// @host           localhost:8080
// @BasePath       /api/v1
// @schemes        http
// @contact.name   API Support
// @contact.email  support@witchslounge.com

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// Parse command line flags
	mode := flag.String("mode", "prod", "server mode (dev/prod)")
	flag.Parse()

	log.Printf("서버 시작 중... (모드: %s)", *mode)

	// 1. Database 초기화
	dbClient, err := bootstrap.SetupDatabase(*mode)
	if err != nil {
		log.Fatalf("데이터베이스 초기화 실패: %v", err)
	}
	defer func(dbClient *ent.Client) {
		_ = dbClient.Close()
	}(dbClient)

	// 2. Redis 초기화
	client, sessionStore, err := bootstrap.SetupRedis(*mode)
	if err != nil {
		log.Fatalf("Redis 초기화 실패: %v", err)
	}
	defer func(client *redis.Client) {
		_ = client.Close()
	}(client)

	// 3. 애플리케이션 의존성 초기화
	deps := bootstrap.SetupAppDependencies(dbClient, sessionStore)

	// 4. Fiber 앱 생성
	errorLogger, err := appLogging.NewErrorLogger("logs/error.log")
	if err != nil {
		log.Fatalf("에러 로그 파일 초기화 실패: %v", err)
	}
	defer func() {
		_ = errorLogger.Close()
	}()

	app := fiber.New(fiber.Config{
		AppName: "Witch's Lounge API",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			status := fiber.StatusInternalServerError
			var fiberErr *fiber.Error
			if errors.As(err, &fiberErr) {
				status = fiberErr.Code
			}

			if status >= fiber.StatusInternalServerError {
				errorLogger.LogHTTPError(c, status, err)
			}

			return c.Status(status).JSON(entity.ErrorResponse{
				Message: "요청 처리 중 오류가 발생했습니다",
				Error:   err.Error(),
			})
		},
	})

	// 5. 미들웨어 설정
	app.Use(middleware.ErrorLoggingMiddleware(errorLogger))

	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} (${latency})\n",
	}))

	// 6. Swagger 문서
	app.Get("/swagger/*", swagger.HandlerDefault)

	// 7. Health check
	app.Get("/api/v1/health", func(c *fiber.Ctx) error {
		return c.JSON(entity.Response{
			Message: "ok",
			Data: fiber.Map{
				"status":      "ok",
				"server_mode": *mode,
				"version":     "1.0.0",
			},
		})
	})

	// 8. 라우터 설정
	v1.SetupRoutes(app, &v1.RouterConfig{
		SessionStore:  deps.SessionStore,
		StoveHandler:  deps.StoveHandler,
		SteamHandler:  deps.SteamHandler,
		UserHandler:   deps.UserHandler,
		RecordHandler: deps.RecordHandler,
		MusicHandler:  deps.MusicHandler,
		StageHandler:  deps.StageHandler,
	})

	// 9. 서버 시작
	log.Printf("서버 준비 완료!")

	if err := app.Listen(":8080"); err != nil {
		log.Fatalf("서버 시작 실패: %v", err)
	}
}
