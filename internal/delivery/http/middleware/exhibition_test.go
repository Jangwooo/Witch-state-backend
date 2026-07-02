package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/ent"
	"github.com/witchs-lounge_backend/internal/domain/entity"
)

const testExhKey = "exh_test_key_1234567890"

// setupExhibitionApp 은 [ExhibitionGate → stubAuth → handler] 체인을 세운다.
// stubAuth 는 c.Locals("user") 가 없으면 401 을 반환하는 최소 인증 스텁 —
// 전시 게이트가 user 를 주입해 우회시키는지 확인하는 용도.
func setupExhibitionApp(cfg ExhibitionGateConfig) *fiber.App {
	app := fiber.New()

	stubAuth := func(c *fiber.Ctx) error {
		// AuthMiddleware 와 동일하게, 전시 요청이면 즉시 통과.
		if IsExhibition(c) {
			return c.Next()
		}
		if c.Locals("user") == nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		return c.Next()
	}

	app.Post("/records", ExhibitionGate(cfg), stubAuth, func(c *fiber.Ctx) error {
		exh := IsExhibition(c)
		usr, _ := c.Locals("user").(*entity.User)
		var uid string
		if usr != nil && usr.User != nil {
			uid = usr.ID.String()
		}
		return c.JSON(fiber.Map{"is_exhibition": exh, "user_id": uid})
	})

	return app
}

func exhibitionUser(id uuid.UUID) *entity.User {
	return &entity.User{User: &ent.User{ID: id}}
}

// 유효한 키가 오면 전시로 통과하고 고정 계정이 주입된다.
func TestExhibitionGate_ValidKeyPasses(t *testing.T) {
	uid := uuid.New()
	cfg := ExhibitionGateConfig{Key: testExhKey, User: exhibitionUser(uid)}
	app := setupExhibitionApp(cfg)

	req := httptest.NewRequest(http.MethodPost, "/records", nil)
	req.Header.Set(ExhibitionHeaderKey, testExhKey)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("valid exhibition key should pass, got %d", resp.StatusCode)
	}
}

// 헤더가 없으면 전시가 아니고, 세션도 없으니 stubAuth 가 401.
func TestExhibitionGate_NoHeaderFallsThroughToAuth(t *testing.T) {
	uid := uuid.New()
	cfg := ExhibitionGateConfig{Key: testExhKey, User: exhibitionUser(uid)}
	app := setupExhibitionApp(cfg)

	req := httptest.NewRequest(http.MethodPost, "/records", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no key should fall through to auth (401), got %d", resp.StatusCode)
	}
}

// 잘못된 키는 전시로 통과하지 않는다 (auth 로 폴백 → 401).
func TestExhibitionGate_WrongKeyRejected(t *testing.T) {
	uid := uuid.New()
	cfg := ExhibitionGateConfig{Key: testExhKey, User: exhibitionUser(uid)}
	app := setupExhibitionApp(cfg)

	req := httptest.NewRequest(http.MethodPost, "/records", nil)
	req.Header.Set(ExhibitionHeaderKey, "wrong_key")
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong key should not pass as exhibition, got %d", resp.StatusCode)
	}
}

// Key 미설정 → 게이트 비활성. 올바른 헤더가 와도 전시로 통과하지 않는다.
func TestExhibitionGate_DisabledWhenNoKey(t *testing.T) {
	uid := uuid.New()
	cfg := ExhibitionGateConfig{Key: "", User: exhibitionUser(uid)}
	app := setupExhibitionApp(cfg)

	req := httptest.NewRequest(http.MethodPost, "/records", nil)
	req.Header.Set(ExhibitionHeaderKey, testExhKey)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("gate must be disabled without key, got %d", resp.StatusCode)
	}
}

// User 미설정(고정 계정 조회 실패) → 게이트 비활성. 키가 맞아도 통과 안 함.
func TestExhibitionGate_DisabledWhenNoUser(t *testing.T) {
	cfg := ExhibitionGateConfig{Key: testExhKey, User: nil}
	app := setupExhibitionApp(cfg)

	req := httptest.NewRequest(http.MethodPost, "/records", nil)
	req.Header.Set(ExhibitionHeaderKey, testExhKey)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("gate must be disabled without fixed user, got %d", resp.StatusCode)
	}
}

// 빈 헤더 값은 빈 키와 매칭되지 않는다 (게이트 활성이어도).
func TestExhibitionGate_EmptyHeaderNeverMatches(t *testing.T) {
	uid := uuid.New()
	cfg := ExhibitionGateConfig{Key: testExhKey, User: exhibitionUser(uid)}
	app := setupExhibitionApp(cfg)

	req := httptest.NewRequest(http.MethodPost, "/records", nil)
	req.Header.Set(ExhibitionHeaderKey, "")
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("empty header should not match, got %d", resp.StatusCode)
	}
}
