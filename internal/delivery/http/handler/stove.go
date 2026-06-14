package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/usecase"
)

// StoveHandler Stove 전용 핸들러
type StoveHandler struct {
	stoveUseCase usecase.StoveUseCase
}

// NewStoveHandler Stove 핸들러 생성자
func NewStoveHandler(stoveUseCase usecase.StoveUseCase) *StoveHandler {
	return &StoveHandler{
		stoveUseCase: stoveUseCase,
	}
}

// SignIn Stove 로그인 처리
// @Summary Stove 플랫폼으로 로그인
// @Description Stove 액세스 토큰 검증 후 로그인 처리 및 세션 생성
// @Tags Stove
// @Accept json
// @Produce json
// @Param body body entity.StoveSignInRequest true "Stove 로그인 요청 정보"
// @Success 200 {object} entity.Response{data=entity.SessionResponse} "로그인 성공 및 세션 정보"
// @Failure 400 {object} entity.ErrorResponse "요청 데이터 검증 실패"
// @Failure 401 {object} entity.ErrorResponse "인증 실패"
// @Failure 500 {object} entity.ErrorResponse "서버 내부 오류"
// @Router /stove/signin [post]
func (h *StoveHandler) SignIn(c *fiber.Ctx) error {
	req := c.Locals("body").(entity.StoveSignInRequest)

	// Stove 로그인 처리
	sessionResp, err := h.stoveUseCase.SignInWithStove(c.Context(), req.AccessToken)
	if err != nil {
		if usecase.IsBannedError(err) {
			return c.Status(fiber.StatusForbidden).JSON(entity.ErrorResponse{
				Message: "차단된 계정입니다.",
				Error:   "account_banned",
				Details: fiber.Map{"reason": "is_banned"},
			})
		}
		if usecase.IsAuthError(err) {
			return c.Status(fiber.StatusUnauthorized).JSON(entity.ErrorResponse{
				Message: "Stove 인증 실패",
				Error:   err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(entity.ErrorResponse{
			Message: "Stove 로그인 처리 실패",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(entity.Response{
		Message: "로그인 성공",
		Data:    sessionResp,
	})
}
