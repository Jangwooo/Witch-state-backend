package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/witchs-lounge_backend/internal/delivery/http/handler"
)

func MusicRouter(router fiber.Router, musicHandler *handler.MusicHandler) {
	router.Get("/musics/actvie", musicHandler.GetActiveMusics)
}
