package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/usecase"
)

type UserHandler struct {
	userUseCase usecase.UserUseCase
}

func NewUserHandler(userUseCase usecase.UserUseCase) *UserHandler {
	return &UserHandler{
		userUseCase: userUseCase,
	}
}

// GetMe 현재 인증된 사용자 정보 반환
// @Summary 현재 사용자 정보 조회
// @Description 세션 정보를 통해 현재 인증된 사용자의 상세 정보를 반환
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} entity.Response{data=entity.UserResponse} "사용자 정보"
// @Failure 401 {object} entity.ErrorResponse "인증 정보 없음"
// @Failure 500 {object} entity.ErrorResponse "서버 내부 오류"
// @Router /user/me [get]
func (h *UserHandler) GetMe(c *fiber.Ctx) error {
	// 미들웨어에서 설정한 사용자 정보 가져오기
	user := c.Locals("user")
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(entity.ErrorResponse{
			Message: "인증 정보를 찾을 수 없습니다",
			Error:   "unauthorized",
		})
	}

	// entity.User 타입으로 변환
	userData, ok := user.(*entity.User)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(entity.ErrorResponse{
			Message: "사용자 정보 형식이 올바르지 않습니다",
			Error:   "invalid_user_context",
		})
	}

	// 사용자 정보 응답
	return c.Status(fiber.StatusOK).JSON(entity.Response{
		Message: "사용자 정보 조회 성공",
		Data:    userData.ToResponse(),
	})
}
