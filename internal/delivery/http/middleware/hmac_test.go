package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/witchs-lounge_backend/ent"
	"github.com/witchs-lounge_backend/ent/user"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/infrastructure/hmacauth"
)

// memoryNonceStore 는 테스트용 NonceStore 입니다. TTL 만료는 무시합니다.
type memoryNonceStore struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func newMemoryNonceStore() *memoryNonceStore {
	return &memoryNonceStore{seen: make(map[string]struct{})}
}

func (m *memoryNonceStore) SetNX(_ context.Context, key string, _ time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.seen[key]; ok {
		return false, nil
	}
	m.seen[key] = struct{}{}
	return true, nil
}

// setupApp 는 테스트용 Fiber 앱을 만들고 /records 경로에 가짜 user/sessionID/hmacSecret 을
// Locals 에 주입한 뒤 HMACMiddleware 를 거쳐 200 응답을 돌려줍니다.
func setupApp(t *testing.T, hcfg *hmacauth.Config, secret string, platformType, platformUserID string, nowFn func() time.Time) (*fiber.App, *memoryNonceStore) {
	t.Helper()
	app := fiber.New()
	nonces := newMemoryNonceStore()

	mw := HMACMiddleware(HMACVerifierConfig{
		Config:     hcfg,
		NonceStore: nonces,
		NowFunc:    nowFn,
	})

	// 사용자/세션 주입 미들웨어 (AuthMiddleware 시뮬레이션).
	app.Use(func(c *fiber.Ctx) error {
		u := &entity.User{User: &ent.User{
			PlatformType:   user.PlatformType(platformType),
			PlatformUserID: platformUserID,
		}}
		c.Locals("user", u)
		c.Locals("sessionID", "test-session-id")
		c.Locals("hmacSecret", secret)
		return c.Next()
	})

	app.Get("/api/v1/records", mw, func(c *fiber.Ctx) error {
		verified, _ := c.Locals("hmac_verified").(bool)
		return c.JSON(fiber.Map{"verified": verified})
	})
	app.Post("/api/v1/records", mw, func(c *fiber.Ctx) error {
		verified, _ := c.Locals("hmac_verified").(bool)
		return c.JSON(fiber.Map{"verified": verified})
	})

	return app, nonces
}

func signGET(secret, pathQuery, nonce string, ts int64) string {
	canonical := hmacauth.BuildCanonical("GET", pathQuery, nonce, strconv.FormatInt(ts, 10), hmacauth.EmptyBodySHA256Hex)
	return hmacauth.Sign(secret, canonical)
}

func signPOST(secret, pathQuery, nonce string, ts int64, body []byte) string {
	canonical := hmacauth.BuildCanonical("POST", pathQuery, nonce, strconv.FormatInt(ts, 10), hmacauth.BodySHA256Hex(body))
	return hmacauth.Sign(secret, canonical)
}

func doReq(t *testing.T, app *fiber.App, req *http.Request) (*http.Response, []byte) {
	t.Helper()
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, b
}

// --- 테스트 ---

func TestHMAC_ShadowAcceptsValidRequest(t *testing.T) {
	cfg := &hmacauth.Config{Mode: hmacauth.ModeShadow}
	now := time.Unix(1750000000, 0)
	app, _ := setupApp(t, cfg, "test-secret-key", "steam", "7656", func() time.Time { return now })

	pathQuery := "/api/v1/records?music_id=test"
	nonce := "9f1a2c3b4d5e6f708192a3b4c5d6e7f8"
	ts := now.Unix()
	sig := signGET("test-secret-key", pathQuery, nonce, ts)

	req := httptest.NewRequest(http.MethodGet, pathQuery, nil)
	req.Header.Set("X-Signature", sig)
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Timestamp", strconv.FormatInt(ts, 10))

	resp, body := doReq(t, app, req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", resp.StatusCode, body)
	}
	var out map[string]bool
	_ = json.Unmarshal(body, &out)
	if !out["verified"] {
		t.Fatalf("expected hmac_verified=true, got %v", out)
	}
}

func TestHMAC_ShadowPassesEvenWhenFails(t *testing.T) {
	cfg := &hmacauth.Config{Mode: hmacauth.ModeShadow}
	app, _ := setupApp(t, cfg, "test-secret-key", "steam", "7656", nil)

	// 헤더 없음 → shadow 모드는 통과해야 함
	req := httptest.NewRequest(http.MethodGet, "/api/v1/records?music_id=test", nil)
	resp, body := doReq(t, app, req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("shadow without headers should return 200, got %d, body=%s", resp.StatusCode, body)
	}
}

func TestHMAC_OffIsNoOp(t *testing.T) {
	cfg := &hmacauth.Config{Mode: hmacauth.ModeOff}
	app, _ := setupApp(t, cfg, "", "steam", "7656", nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/records?music_id=test", nil)
	resp, body := doReq(t, app, req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("off mode should pass, got %d body=%s", resp.StatusCode, body)
	}
	var out map[string]bool
	_ = json.Unmarshal(body, &out)
	if out["verified"] {
		t.Fatal("off mode should not set hmac_verified=true")
	}
}

func TestHMAC_EnforceRejectsTimestampSkew(t *testing.T) {
	cfg := &hmacauth.Config{
		Mode: hmacauth.ModeEnforce,
		EnforcePlatformIDs: map[string]struct{}{
			"steam:7656": {},
		},
	}
	now := time.Unix(1750000000, 0)
	app, _ := setupApp(t, cfg, "test-secret-key", "steam", "7656", func() time.Time { return now })

	pathQuery := "/api/v1/records?music_id=test"
	nonce := "9f1a2c3b4d5e6f708192a3b4c5d6e7f8"
	// 10분 전 (skew window 5분 초과)
	ts := now.Unix() - 600
	sig := signGET("test-secret-key", pathQuery, nonce, ts)

	req := httptest.NewRequest(http.MethodGet, pathQuery, nil)
	req.Header.Set("X-Signature", sig)
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Timestamp", strconv.FormatInt(ts, 10))

	resp, body := doReq(t, app, req)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for skewed timestamp, got %d body=%s", resp.StatusCode, body)
	}
}

func TestHMAC_EnforceRejectsNonceReplay(t *testing.T) {
	cfg := &hmacauth.Config{
		Mode:               hmacauth.ModeEnforce,
		EnforcePlatformIDs: map[string]struct{}{"steam:7656": {}},
	}
	now := time.Unix(1750000000, 0)
	app, _ := setupApp(t, cfg, "test-secret-key", "steam", "7656", func() time.Time { return now })

	pathQuery := "/api/v1/records?music_id=test"
	nonce := "9f1a2c3b4d5e6f708192a3b4c5d6e7f8"
	ts := now.Unix()
	sig := signGET("test-secret-key", pathQuery, nonce, ts)

	mkReq := func() *http.Request {
		req := httptest.NewRequest(http.MethodGet, pathQuery, nil)
		req.Header.Set("X-Signature", sig)
		req.Header.Set("X-Nonce", nonce)
		req.Header.Set("X-Timestamp", strconv.FormatInt(ts, 10))
		return req
	}

	resp, _ := doReq(t, app, mkReq())
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first request should succeed, got %d", resp.StatusCode)
	}
	resp, body := doReq(t, app, mkReq())
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("replayed nonce should 401, got %d body=%s", resp.StatusCode, body)
	}
}

func TestHMAC_EnforceRejectsBodyTamper(t *testing.T) {
	cfg := &hmacauth.Config{
		Mode:               hmacauth.ModeEnforce,
		EnforcePlatformIDs: map[string]struct{}{"steam:7656": {}},
	}
	now := time.Unix(1750000000, 0)
	app, _ := setupApp(t, cfg, "test-secret-key", "steam", "7656", func() time.Time { return now })

	pathQuery := "/api/v1/records"
	nonce := "9f1a2c3b4d5e6f708192a3b4c5d6e7f8"
	ts := now.Unix()
	originalBody := []byte(`{"score":100}`)
	sig := signPOST("test-secret-key", pathQuery, nonce, ts, originalBody)

	// 변조된 body 로 요청
	tampered := []byte(`{"score":9999}`)
	req := httptest.NewRequest(http.MethodPost, pathQuery, bytes.NewReader(tampered))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature", sig)
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Timestamp", strconv.FormatInt(ts, 10))

	resp, body := doReq(t, app, req)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("body tamper should 401, got %d body=%s", resp.StatusCode, body)
	}
}

func TestHMAC_EnforcePassesNonWhitelistedUser(t *testing.T) {
	// enforce 모드여도 화이트리스트 밖 유저는 검증 실패해도 통과 (= 일반 유저 무영향)
	cfg := &hmacauth.Config{
		Mode:               hmacauth.ModeEnforce,
		EnforcePlatformIDs: map[string]struct{}{"steam:7656": {}},
	}
	app, _ := setupApp(t, cfg, "test-secret-key", "steam", "9999", nil)

	// 헤더 누락 → 검증 실패하지만 화이트리스트 밖이라 통과해야 함
	req := httptest.NewRequest(http.MethodGet, "/api/v1/records?music_id=test", nil)
	resp, body := doReq(t, app, req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("non-whitelisted user should pass even on validation fail, got %d body=%s", resp.StatusCode, body)
	}
}

func TestHMAC_GraceForSessionWithoutSecret(t *testing.T) {
	// 구버전 세션(hmac_secret 비어있음) + 화이트리스트 밖이라 통과
	cfg := &hmacauth.Config{Mode: hmacauth.ModeShadow}
	app, _ := setupApp(t, cfg, "", "steam", "7656", nil)

	now := time.Unix(1750000000, 0)
	pathQuery := "/api/v1/records?music_id=test"
	nonce := "9f1a2c3b4d5e6f708192a3b4c5d6e7f8"
	ts := now.Unix()
	// 클라가 헤더를 보냈지만 세션에 시크릿 없음 → shadow 라 200 + verified=false
	req := httptest.NewRequest(http.MethodGet, pathQuery, nil)
	req.Header.Set("X-Signature", "00")
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Timestamp", strconv.FormatInt(ts, 10))

	resp, _ := doReq(t, app, req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("shadow with missing secret should pass, got %d", resp.StatusCode)
	}
}
