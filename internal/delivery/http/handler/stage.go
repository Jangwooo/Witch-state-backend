package handler

import (
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/usecase"

	"github.com/gofiber/fiber/v2"
)

type StageHandler struct {
	stageUseCase usecase.StageUseCase
}

func NewStageHandler(stageUseCase usecase.StageUseCase) *StageHandler {
	return &StageHandler{stageUseCase: stageUseCase}
}

// GetStagesByMusicID godoc
// @Summary 음악 id에 따른 스테이지들을 가져옵니다
// @Description 음악 id에 따른 스테이지들을 가져옵니다
// @Tags Stages
// @Accept json
// @Produce json
// @Param music_id path string true "Music ID"
// @Success 200 {object} entity.Response{data=[]dto.StageResponse} "성공"
// @Failure 400 {object} entity.ErrorResponse "잘못된 요청"
// @Failure 403 {object} entity.ErrorResponse "음악이 비활성화됨"
// @Failure 500 {object} entity.ErrorResponse "서버 에러"
// @Router /musics/{music_id}/stages [get]
func (h *StageHandler) GetStagesByMusicID(c *fiber.Ctx) error {
	musicID := c.Params("music_id")
	if musicID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(entity.ErrorResponse{
			Message: "Music ID is required",
		})
	}

	stages, err := h.stageUseCase.GetStagesByMusicID(c.Context(), musicID)
	if err != nil {
		if err.Error() == "music is not active" {
			return c.Status(fiber.StatusForbidden).JSON(entity.ErrorResponse{
				Message: "Music is not active",
				Error:   err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(entity.ErrorResponse{
			Message: "Failed to get stages",
			Error:   err.Error(),
		})
	}

	return c.JSON(entity.Response{
		Message: "Stages retrieved successfully",
		Data:    stages,
	})
}
