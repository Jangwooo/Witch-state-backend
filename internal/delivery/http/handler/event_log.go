package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/usecase"
)

type EventLogHandler struct {
	eventLogUseCase usecase.EventLogUseCase
}

func NewEventLogHandler(eventLogUseCase usecase.EventLogUseCase) *EventLogHandler {
	return &EventLogHandler{eventLogUseCase: eventLogUseCase}
}

// CreateEventLog Event 상태변경 로그 적재 (단건)
// @Summary Event 상태변경 로그 적재 (단건)
// @Description Lobby Event 상태변경을 서버에 로그로 적재합니다 (멱등성: client_log_id)
// @Tags EventLog
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body entity.CreateEventLogRequest true "이벤트 로그 요청"
// @Success 200 {object} entity.Response{data=repository.EventLogResultItem} "처리 결과"
// @Failure 400 {object} entity.ErrorResponse "잘못된 요청 형식"
// @Failure 401 {object} entity.ErrorResponse "인증 필요"
// @Failure 500 {object} entity.ErrorResponse "서버 내부 오류"
// @Router /events/log [post]
func (h *EventLogHandler) CreateEventLog(c *fiber.Ctx) error {
	userEntity, errResp := eventLogUser(c)
	if errResp != nil {
		return errResp
	}

	var req entity.CreateEventLogRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(entity.ErrorResponse{Message: "요청 파싱 실패", Error: err.Error()})
	}

	resp, err := h.eventLogUseCase.Create(c.Context(), userEntity.ID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(entity.ErrorResponse{Message: "이벤트 로그 처리 실패", Error: err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(entity.Response{
		Message: "이벤트 로그 적재 성공",
		Data:    resp,
	})
}

// CreateEventLogsBatch Event 상태변경 로그 일괄 적재
// @Summary Event 상태변경 로그 일괄 적재
// @Description 오프라인 누적된 Event 상태변경 로그를 일괄 적재합니다 (멱등성: client_log_id)
// @Tags EventLog
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body entity.BatchEventLogsRequest true "batch 요청"
// @Success 200 {object} entity.Response{data=repository.BatchEventLogsResponse} "처리 결과"
// @Failure 400 {object} entity.ErrorResponse "잘못된 요청 형식 / 100건 초과"
// @Failure 401 {object} entity.ErrorResponse "인증 필요"
// @Failure 500 {object} entity.ErrorResponse "서버 내부 오류"
// @Router /events/log/batch [post]
func (h *EventLogHandler) CreateEventLogsBatch(c *fiber.Ctx) error {
	userEntity, errResp := eventLogUser(c)
	if errResp != nil {
		return errResp
	}

	var req entity.BatchEventLogsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(entity.ErrorResponse{Message: "요청 파싱 실패", Error: err.Error()})
	}
	if len(req.Logs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(entity.ErrorResponse{Message: "logs 배열이 비어 있습니다", Error: "empty_logs"})
	}
	if len(req.Logs) > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(entity.ErrorResponse{Message: "logs 항목 수가 상한(100)을 초과했습니다", Error: "too_many_logs"})
	}

	resp, err := h.eventLogUseCase.CreateBatch(c.Context(), userEntity.ID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(entity.ErrorResponse{Message: "batch 처리 실패", Error: err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(entity.Response{
		Message: "이벤트 로그 일괄 적재 성공",
		Data:    resp,
	})
}

// eventLogUser c.Locals("user") 에서 인증 유저를 꺼낸다. 실패 시 즉시 응답할 error 를 반환.
func eventLogUser(c *fiber.Ctx) (*entity.User, error) {
	usr := c.Locals("user")
	if usr == nil {
		return nil, c.Status(fiber.StatusUnauthorized).JSON(entity.ErrorResponse{
			Message: "인증 필요",
			Error:   "unauthorized",
		})
	}
	userEntity, ok := usr.(*entity.User)
	if !ok {
		return nil, c.Status(fiber.StatusInternalServerError).JSON(entity.ErrorResponse{
			Message: "사용자 정보 형식 오류",
			Error:   "invalid_user_context",
		})
	}
	return userEntity, nil
}
