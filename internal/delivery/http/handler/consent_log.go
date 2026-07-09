package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/infrastructure/session"
	"github.com/witchs-lounge_backend/internal/usecase"
)

// ConsentLogHandler EULA/개인정보 동의 로그 적재 핸들러.
// 무인증 라우트 — auth 미들웨어를 안 태우므로, 세션(Bearer)이 있으면 핸들러가 직접 조회해
// user_id 를 채운다(옵션). 없으면 user_id 는 nil 로 저장(로그인 전 발생).
type ConsentLogHandler struct {
	consentLogUseCase usecase.ConsentLogUseCase
	sessionStore      session.SessionStore
}

func NewConsentLogHandler(consentLogUseCase usecase.ConsentLogUseCase, sessionStore session.SessionStore) *ConsentLogHandler {
	return &ConsentLogHandler{consentLogUseCase: consentLogUseCase, sessionStore: sessionStore}
}

// CreateConsentLog 동의 로그 적재 (단건)
// @Summary 동의 로그 적재 (단건)
// @Description EULA/개인정보 동의를 서버에 적재합니다 (무인증, 멱등성: client_consent_id)
// @Tags ConsentLog
// @Accept json
// @Produce json
// @Param body body entity.CreateConsentLogRequest true "동의 로그 요청"
// @Success 200 {object} entity.Response{data=repository.ConsentLogResultItem} "처리 결과"
// @Failure 400 {object} entity.ErrorResponse "잘못된 요청 형식"
// @Failure 500 {object} entity.ErrorResponse "서버 내부 오류"
// @Router /consents/log [post]
func (h *ConsentLogHandler) CreateConsentLog(c *fiber.Ctx) error {
	var req entity.CreateConsentLogRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(entity.ErrorResponse{Message: "요청 파싱 실패", Error: err.Error()})
	}

	userID := h.optionalUserID(c)

	resp, err := h.consentLogUseCase.Create(c.Context(), userID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(entity.ErrorResponse{Message: "동의 로그 처리 실패", Error: err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(entity.Response{
		Message: "동의 로그 적재 성공",
		Data:    resp,
	})
}

// CreateConsentLogsBatch 동의 로그 일괄 적재
// @Summary 동의 로그 일괄 적재
// @Description 오프라인 누적된 동의 로그를 일괄 적재합니다 (무인증, 멱등성: client_consent_id)
// @Tags ConsentLog
// @Accept json
// @Produce json
// @Param body body entity.BatchConsentLogsRequest true "batch 요청"
// @Success 200 {object} entity.Response{data=repository.BatchConsentLogsResponse} "처리 결과"
// @Failure 400 {object} entity.ErrorResponse "잘못된 요청 형식 / 100건 초과"
// @Failure 500 {object} entity.ErrorResponse "서버 내부 오류"
// @Router /consents/log/batch [post]
func (h *ConsentLogHandler) CreateConsentLogsBatch(c *fiber.Ctx) error {
	var req entity.BatchConsentLogsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(entity.ErrorResponse{Message: "요청 파싱 실패", Error: err.Error()})
	}
	if len(req.Consents) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(entity.ErrorResponse{Message: "consents 배열이 비어 있습니다", Error: "empty_consents"})
	}
	if len(req.Consents) > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(entity.ErrorResponse{Message: "consents 항목 수가 상한(100)을 초과했습니다", Error: "too_many_consents"})
	}

	userID := h.optionalUserID(c)

	resp, err := h.consentLogUseCase.CreateBatch(c.Context(), userID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(entity.ErrorResponse{Message: "batch 처리 실패", Error: err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(entity.Response{
		Message: "동의 로그 일괄 적재 성공",
		Data:    resp,
	})
}

// optionalUserID 는 Authorization: Bearer 세션이 있으면 그 user_id 를 반환한다.
// 무인증 라우트이므로 헤더가 없거나 세션이 무효여도 에러 없이 nil 을 반환한다(그대로 통과).
func (h *ConsentLogHandler) optionalUserID(c *fiber.Ctx) *uuid.UUID {
	if h.sessionStore == nil {
		return nil
	}
	authHeader := c.Get("Authorization")
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		return nil
	}
	sessionID := strings.TrimPrefix(authHeader, bearerPrefix)
	user, err := h.sessionStore.Get(c.Context(), sessionID)
	if err != nil || user == nil || user.User == nil {
		return nil // 무인증 라우트 — 세션 무효여도 통과, user_id 만 비움
	}
	id := user.ID
	return &id
}
