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
// @Description Stove 플랫폼 사용자 정보로 로그인 처리 및 세션 생성
// @Tags Stove
// @Accept json
// @Produce json
// @Param body body entity.StoveSignInRequest true "Stove 로그인 요청 정보"
// @Success 200 {object} entity.SessionResponse "로그인 성공 및 세션 정보"
// @Failure 400 {object} entity.ErrorResponse "요청 데이터 검증 실패"
// @Failure 500 {object} entity.ErrorResponse "서버 내부 오류"
// @Router /stove/signin [post]
func (h *StoveHandler) SignIn(c *fiber.Ctx) error {
	req := c.Locals("body").(entity.StoveSignInRequest)

	// Stove 로그인 처리
	sessionResp, err := h.stoveUseCase.SignInWithStove(c.Context(), struct {
		ID          string
		Email       string
		AvatarUrl   string
		DisplayName string
	}{
		ID:          req.ID,
		Email:       req.Email,
		AvatarUrl:   req.AvatarUrl,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(sessionResp)
}
