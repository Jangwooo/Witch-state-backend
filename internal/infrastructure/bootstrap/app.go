package bootstrap

import (
	"github.com/witchs-lounge_backend/ent"
	"github.com/witchs-lounge_backend/internal/delivery/http/handler"
	"github.com/witchs-lounge_backend/internal/infrastructure/session"
	"github.com/witchs-lounge_backend/internal/repository"
	"github.com/witchs-lounge_backend/internal/usecase"
)

// AppDependencies 애플리케이션 의존성
type AppDependencies struct {
	// Handlers
	StoveHandler  *handler.StoveHandler
	UserHandler   *handler.UserHandler
	RecordHandler *handler.RecordHandler

	// Session
	SessionStore session.SessionStore
}

// SetupAppDependencies 애플리케이션 의존성 초기화
func SetupAppDependencies(dbClient *ent.Client, sessionStore session.SessionStore) *AppDependencies {
	// Initialize repositories
	userRepo := repository.NewUserRepository(dbClient)
	recordRepo := repository.NewRecordRepository(dbClient)

	// Initialize use cases
	stoveUseCase := usecase.NewStoveUseCase(userRepo, sessionStore)
	userUseCase := usecase.NewUserUseCase(userRepo)
	recordUseCase := usecase.NewRecordUseCase(recordRepo)

	// Initialize handlers
	stoveHandler := handler.NewStoveHandler(stoveUseCase)
	userHandler := handler.NewUserHandler(userUseCase)
	recordHandler := handler.NewRecordHandler(recordUseCase)

	return &AppDependencies{
		StoveHandler:  stoveHandler,
		UserHandler:   userHandler,
		RecordHandler: recordHandler,
		SessionStore:  sessionStore,
	}
}
