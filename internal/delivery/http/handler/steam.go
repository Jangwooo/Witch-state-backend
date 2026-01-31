package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/usecase"
)

type SteamHandler struct {
	steamUseCase usecase.SteamUseCase
}

func NewSteamHandler(steamUseCase usecase.SteamUseCase) *SteamHandler {
	return &SteamHandler{
		steamUseCase: steamUseCase,
	}
}

// SignIn Steam 로그인 처리
// @Summary Steam 로그인
// @Description Steam 티켓 인증을 통해 로그인 처리 및 세션 생성
// @Tags Steam
// @Accept json
// @Produce json
// @Param body body entity.SteamSignInRequest true "Steam 로그인 요청 정보"
// @Success 200 {object} entity.Response{data=entity.SessionResponse} "로그인 성공 및 세션 정보"
// @Failure 401 {object} entity.ErrorResponse "인증 실패"
// @Failure 500 {object} entity.ErrorResponse "서버 내부 오류"
// @Router /steam/signin [post]
func (h *SteamHandler) SignIn(c *fiber.Ctx) error {
	req := c.Locals("body").(entity.SteamSignInRequest)

	sessionResp, err := h.steamUseCase.SignInWithSteam(c.Context(), req.Id, req.Ticket)
	if err != nil {
		if usecase.IsAuthError(err) {
			return c.Status(fiber.StatusUnauthorized).JSON(entity.ErrorResponse{
				Message: "Steam 인증 실패",
				Error:   err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(entity.ErrorResponse{
			Message: "Steam 로그인 처리 실패",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(entity.Response{
		Message: "로그인 성공",
		Data:    sessionResp,
	})
}
