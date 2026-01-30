package handler

import (
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/usecase"

	"github.com/gofiber/fiber/v2"
)

type MusicHandler struct {
	musicUseCase usecase.MusicUseCase
}

func NewMusicHandler(musicUseCase usecase.MusicUseCase) *MusicHandler {
	return &MusicHandler{musicUseCase: musicUseCase}
}

// GetActiveMusics godoc
// @Summary 활성화 된 악곡들을 가져옵니다
// @Description 활성화 된 악곡들을 가져옵니다
// @Tags Musics
// @Accept json
// @Produce json
// @Success 200 {object} entity.Response{data=[]dto.MusicResponse} "성공"
// @Failure 500 {object} entity.ErrorResponse "서버 에러"
// @Router /musics [get]
func (h *MusicHandler) GetActiveMusics(c *fiber.Ctx) error {
	musics, err := h.musicUseCase.GetActiveMusics(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(entity.ErrorResponse{
			Message: "Failed to get active musics",
			Error:   err.Error(),
		})
	}

	return c.JSON(entity.Response{
		Message: "Active musics retrieved successfully",
		Data:    musics,
	})
}
