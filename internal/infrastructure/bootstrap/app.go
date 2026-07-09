package bootstrap

import (
	"context"

	"github.com/witchs-lounge_backend/ent"
	"github.com/witchs-lounge_backend/internal/delivery/http/handler"
	"github.com/witchs-lounge_backend/internal/delivery/http/middleware"
	domainrepo "github.com/witchs-lounge_backend/internal/domain/repository"
	"github.com/witchs-lounge_backend/internal/infrastructure/hmacauth"
	"github.com/witchs-lounge_backend/internal/infrastructure/logging"
	"github.com/witchs-lounge_backend/internal/infrastructure/session"
	"github.com/witchs-lounge_backend/internal/repository"
	"github.com/witchs-lounge_backend/internal/usecase"
)

// AppDependencies 애플리케이션 의존성
type AppDependencies struct {
	// Handlers
	StoveHandler  *handler.StoveHandler
	SteamHandler  *handler.SteamHandler
	UserHandler       *handler.UserHandler
	RecordHandler     *handler.RecordHandler
	EventLogHandler   *handler.EventLogHandler
	ConsentLogHandler *handler.ConsentLogHandler
	MusicHandler      *handler.MusicHandler
	StageHandler      *handler.StageHandler

	// Session
	SessionStore session.SessionStore

	// HMAC 검증 설정 (HMAC_MODE, 화이트리스트, PCT)
	HMACConfig *hmacauth.Config

	// users.is_banned 확인용
	UserRepo domainrepo.UserRepository
	BanCache *middleware.BanCache

	// 전시(EXHIBITION) 게이트 설정 (EXHIBITION_KEY + 고정 계정). 미설정 시 게이트 비활성.
	ExhibitionGate middleware.ExhibitionGateConfig
}

// SetupAppDependencies 애플리케이션 의존성 초기화
func SetupAppDependencies(dbClient *ent.Client, sessionStore session.SessionStore, errorLogger *logging.ErrorLogger) *AppDependencies {
	// Initialize repositories
	userRepo := repository.NewUserRepository(dbClient)
	recordRepo := repository.NewRecordRepository(dbClient)
	eventLogRepo := repository.NewEventLogRepository(dbClient)
	consentLogRepo := repository.NewConsentLogRepository(dbClient)
	musicRepo := repository.NewMusicRepository(dbClient)
	stageRepo := repository.NewStageRepository(dbClient)
	progressionRepo := repository.NewProgressionRepository(dbClient)

	// HMAC 검증 설정 로드. 미설정 시 ModeOff = 현재 운영 동작과 동일.
	hmacCfg := hmacauth.LoadConfig()

	// 전시(EXHIBITION) 게이트 설정 로드. EXHIBITION_KEY/EXHIBITION_USER_ID 미설정 시 비활성 (기존 동작 불변).
	exhibitionGate := middleware.LoadExhibitionGate(context.Background(), userRepo)

	// Sanity validator 운영 모드 로드. 미설정 시 ModeOff = 100% 기존 운영 동작.
	sanityMode := usecase.LoadSanityMode()
	sanityValidator := usecase.NewSanityValidator(sanityMode, errorLogger)

	// Initialize use cases
	stoveUseCase := usecase.NewStoveUseCase(userRepo, sessionStore, hmacCfg)
	steamUseCase := usecase.NewSteamUseCase(userRepo, sessionStore, hmacCfg)
	userUseCase := usecase.NewUserUseCase(userRepo)
	recordUseCase := usecase.NewRecordUseCase(recordRepo, userRepo, musicRepo, stageRepo, progressionRepo, sanityValidator, sanityMode, errorLogger)
	eventLogUseCase := usecase.NewEventLogUseCase(eventLogRepo, errorLogger)
	consentLogUseCase := usecase.NewConsentLogUseCase(consentLogRepo, errorLogger)
	musicUseCase := usecase.NewMusicUseCase(musicRepo)
	stageUseCase := usecase.NewStageUseCase(stageRepo, musicRepo)

	// Initialize handlers
	stoveHandler := handler.NewStoveHandler(stoveUseCase)
	steamHandler := handler.NewSteamHandler(steamUseCase)
	userHandler := handler.NewUserHandler(userUseCase)
	recordHandler := handler.NewRecordHandler(recordUseCase)
	eventLogHandler := handler.NewEventLogHandler(eventLogUseCase)
	consentLogHandler := handler.NewConsentLogHandler(consentLogUseCase, sessionStore)
	musicHandler := handler.NewMusicHandler(musicUseCase)
	stageHandler := handler.NewStageHandler(stageUseCase)

	return &AppDependencies{
		StoveHandler:  stoveHandler,
		SteamHandler:  steamHandler,
		UserHandler:       userHandler,
		RecordHandler:     recordHandler,
		EventLogHandler:   eventLogHandler,
		ConsentLogHandler: consentLogHandler,
		MusicHandler:      musicHandler,
		StageHandler:      stageHandler,

		SessionStore:   sessionStore,
		HMACConfig:     hmacCfg,
		UserRepo:       userRepo,
		BanCache:       middleware.NewBanCache(0), // 기본 TTL = 60s
		ExhibitionGate: exhibitionGate,
	}
}
