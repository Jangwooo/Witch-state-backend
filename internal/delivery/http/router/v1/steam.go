package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/witchs-lounge_backend/internal/delivery/http/handler"
	"github.com/witchs-lounge_backend/internal/delivery/http/middleware"
	"github.com/witchs-lounge_backend/internal/domain/entity"
)

func NewSteamRouter(router fiber.Router, steamHandler *handler.SteamHandler) {
	steam := router.Group("/steam")

	steam.Post("/signin", middleware.ValidateBody[entity.SteamSignInRequest](), steamHandler.SignIn)
}
