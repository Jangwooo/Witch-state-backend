package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/witchs-lounge_backend/internal/delivery/http/handler"
)

func StageRouter(router fiber.Router, stageHandler *handler.StageHandler) {
	router.Get("/musics/:music_id/stages", stageHandler.GetStagesByMusicID)
}
