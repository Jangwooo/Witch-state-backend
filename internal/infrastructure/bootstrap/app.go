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
	SteamHandler  *handler.SteamHandler
	UserHandler   *handler.UserHandler
	RecordHandler *handler.RecordHandler
	MusicHandler  *handler.MusicHandler
	StageHandler  *handler.StageHandler

	// Session
	SessionStore session.SessionStore
}

// SetupAppDependencies 애플리케이션 의존성 초기화
func SetupAppDependencies(dbClient *ent.Client, sessionStore session.SessionStore) *AppDependencies {
	// Initialize repositories
	userRepo := repository.NewUserRepository(dbClient)
	recordRepo := repository.NewRecordRepository(dbClient)
	musicRepo := repository.NewMusicRepository(dbClient)
	stageRepo := repository.NewStageRepository(dbClient)

	// Initialize use cases
	stoveUseCase := usecase.NewStoveUseCase(userRepo, sessionStore)
	steamUseCase := usecase.NewSteamUseCase(userRepo, sessionStore)
	userUseCase := usecase.NewUserUseCase(userRepo)
	recordUseCase := usecase.NewRecordUseCase(recordRepo)
	musicUseCase := usecase.NewMusicUseCase(musicRepo)
	stageUseCase := usecase.NewStageUseCase(stageRepo, musicRepo)

	// Initialize handlers
	stoveHandler := handler.NewStoveHandler(stoveUseCase)
	steamHandler := handler.NewSteamHandler(steamUseCase)
	userHandler := handler.NewUserHandler(userUseCase)
	recordHandler := handler.NewRecordHandler(recordUseCase)
	musicHandler := handler.NewMusicHandler(musicUseCase)
	stageHandler := handler.NewStageHandler(stageUseCase)

	return &AppDependencies{
		StoveHandler:  stoveHandler,
		SteamHandler:  steamHandler,
		UserHandler:   userHandler,
		RecordHandler: recordHandler,
		MusicHandler:  musicHandler,
		StageHandler:  stageHandler,

		SessionStore: sessionStore,
	}
}
