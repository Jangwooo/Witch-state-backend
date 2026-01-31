package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/usecase"
)

type RecordHandler struct {
	recordUseCase usecase.RecordUseCase
}

func NewRecordHandler(recordUseCase usecase.RecordUseCase) *RecordHandler {
	return &RecordHandler{recordUseCase: recordUseCase}
}

// CreateRecord 플레이 기록 저장
// @Summary 플레이 기록 저장
// @Description 플레이 기록을 저장합니다
// @Tags Record
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body entity.CreateRecordRequest true "플레이 기록 요청"
// @Success 200 {object} entity.Response{data=repository.SingleRecordResponse} "생성된 기록"
// @Failure 400 {object} entity.ErrorResponse "잘못된 요청 형식"
// @Failure 401 {object} entity.ErrorResponse "인증 필요"
// @Failure 500 {object} entity.ErrorResponse "서버 내부 오류"
// @Router /records [post]
func (h *RecordHandler) CreateRecord(c *fiber.Ctx) error {
	usr := c.Locals("user")
	if usr == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(entity.ErrorResponse{
			Message: "인증 필요",
			Error:   "unauthorized",
		})
	}
	userEntity, ok := usr.(*entity.User)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(entity.ErrorResponse{
			Message: "사용자 정보 형식 오류",
			Error:   "invalid_user_context",
		})
	}

	var req entity.CreateRecordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(entity.ErrorResponse{Message: "요청 파싱 실패", Error: err.Error()})
	}

	resp, err := h.recordUseCase.Create(c.Context(), userEntity.ID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(entity.ErrorResponse{Message: "기록 처리 실패", Error: err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(entity.Response{
		Message: "기록 저장 성공",
		Data:    resp,
	})
}

// ListRecords 전체 기록 조회
// @Summary 전체 기록 조회
// @Description 해당 곡/스테이지의 전체 기록을 조회합니다
// @Tags Record
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param music_id query string true "음악 ID(UUID)"
// @Param stage_id query string true "스테이지 ID(UUID)"
// @Success 200 {object} entity.Response{data=repository.ListRecordResponse} "전체 기록"
// @Failure 400 {object} entity.ErrorResponse
// @Failure 401 {object} entity.ErrorResponse
// @Failure 500 {object} entity.ErrorResponse
// @Router /records [get]
func (h *RecordHandler) ListRecords(c *fiber.Ctx) error {
	usr := c.Locals("user")
	if usr == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(entity.ErrorResponse{
			Message: "인증 필요",
			Error:   "unauthorized",
		})
	}
	userEntity, ok := usr.(*entity.User)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(entity.ErrorResponse{
			Message: "사용자 정보 형식 오류",
			Error:   "invalid_user_context",
		})
	}

	musicID, stageID := c.Query("music_id"), c.Query("stage_id")

	resp, err := h.recordUseCase.List(c.Context(), userEntity.ID, musicID, stageID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(entity.ErrorResponse{Message: "기록 조회 실패", Error: err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(entity.Response{
		Message: "기록 조회 성공",
		Data:    resp,
	})
}

// BestRecord 최고 기록 조회
// @Summary 최고 기록 조회
// @Description 해당 곡/스테이지의 최고 기록을 조회합니다
// @Tags Record
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param music_id query string true "음악 ID(UUID)"
// @Param stage_id query string true "스테이지 ID(UUID)"
// @Success 200 {object} entity.Response{data=repository.BestRecordResponse} "최고 기록"
// @Failure 400 {object} entity.ErrorResponse
// @Failure 401 {object} entity.ErrorResponse
// @Failure 500 {object} entity.ErrorResponse
// @Router /records/best [get]
func (h *RecordHandler) BestRecord(c *fiber.Ctx) error {
	usr := c.Locals("user")
	if usr == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(entity.ErrorResponse{
			Message: "인증 필요",
			Error:   "unauthorized",
		})
	}
	userEntity, ok := usr.(*entity.User)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(entity.ErrorResponse{
			Message: "사용자 정보 형식 오류",
			Error:   "invalid_user_context",
		})
	}

	musicID, stageID := c.Query("music_id"), c.Query("stage_id")
	if musicID == "" || stageID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(entity.ErrorResponse{Message: "쿼리 파라미터 누락", Error: "music_id, stage_id 필요"})
	}

	resp, err := h.recordUseCase.Best(c.Context(), userEntity.ID, musicID, stageID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(entity.ErrorResponse{Message: "최고 기록 조회 실패", Error: err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(entity.Response{
		Message: "최고 기록 조회 성공",
		Data:    resp,
	})
}

// parseUUID2 보조 함수
// 실제 UUID 변환은 usecase에서 하기 때문에, 여기서는 문자열만 전달하도록 변경
