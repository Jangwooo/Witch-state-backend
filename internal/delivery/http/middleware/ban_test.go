package middleware

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/ent"
	"github.com/witchs-lounge_backend/ent/user"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	domainrepo "github.com/witchs-lounge_backend/internal/domain/repository"
)

// fakeUserRepo 는 도메인 UserRepository 인터페이스를 충족하는 인메모리 fake.
// 본 테스트는 FindByID 만 사용. 나머지 메서드는 panic — 의도된 호출 외 발생 시 즉시 발견.
type fakeUserRepo struct {
	users     map[uuid.UUID]*entity.User
	findCalls atomic.Int32
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: make(map[uuid.UUID]*entity.User)}
}

func (f *fakeUserRepo) FindByID(_ context.Context, id uuid.UUID) (*entity.User, error) {
	f.findCalls.Add(1)
	u, ok := f.users[id]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (f *fakeUserRepo) FindByPlatformUserID(context.Context, string, string) (*entity.User, error) {
	panic("not used")
}
func (f *fakeUserRepo) Create(context.Context, *entity.CreateUserRequest) (*entity.User, error) {
	panic("not used")
}
func (f *fakeUserRepo) UpdatePlatformProfile(context.Context, uuid.UUID, string, string, string, map[string]interface{}) (*entity.User, error) {
	panic("not used")
}
func (f *fakeUserRepo) UpdateLastLogin(context.Context, uuid.UUID, time.Time) (*entity.User, error) {
	panic("not used")
}
func (f *fakeUserRepo) UpdateExpAndLevel(context.Context, uuid.UUID, int, int) (*entity.User, error) {
	panic("not used")
}

// compile-time check
var _ domainrepo.UserRepository = (*fakeUserRepo)(nil)

func setupBanApp(t *testing.T, userID uuid.UUID, repo *fakeUserRepo, cache *BanCache) *fiber.App {
	t.Helper()
	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		u := &entity.User{User: &ent.User{
			ID:             userID,
			PlatformType:   user.PlatformType("steam"),
			PlatformUserID: "1",
		}}
		c.Locals("user", u)
		return c.Next()
	})

	app.Get("/protected", BanCheckMiddleware(repo, cache), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	return app
}

func TestBan_AllowsActiveUser(t *testing.T) {
	id := uuid.New()
	repo := newFakeUserRepo()
	repo.users[id] = &entity.User{User: &ent.User{ID: id, IsBanned: false}}
	app := setupBanApp(t, id, repo, NewBanCache(time.Minute))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("active user should pass, got %d", resp.StatusCode)
	}
}

func TestBan_RejectsBannedUserWith403(t *testing.T) {
	id := uuid.New()
	repo := newFakeUserRepo()
	repo.users[id] = &entity.User{User: &ent.User{ID: id, IsBanned: true}}
	app := setupBanApp(t, id, repo, NewBanCache(time.Minute))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	resp, body := readResp(t, app, req)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("banned user should 403, got %d", resp.StatusCode)
	}
	var payload map[string]any
	_ = json.Unmarshal(body, &payload)
	if payload["error"] != "account_banned" {
		t.Fatalf("expected error=account_banned, got %v", payload)
	}
	if details, ok := payload["details"].(map[string]any); !ok || details["reason"] != "is_banned" {
		t.Fatalf("expected details.reason=is_banned, got %v", payload["details"])
	}
}

func TestBan_CacheHitSkipsDB(t *testing.T) {
	id := uuid.New()
	repo := newFakeUserRepo()
	repo.users[id] = &entity.User{User: &ent.User{ID: id, IsBanned: false}}
	cache := NewBanCache(time.Minute)
	app := setupBanApp(t, id, repo, cache)

	// 1차 요청 → DB 조회 1회 + 캐시 적재
	resp, _ := app.Test(httptest.NewRequest(http.MethodGet, "/protected", nil), -1)
	resp.Body.Close()

	// 2~5차 요청 → 캐시 hit, DB 조회 추가 안 함
	for i := 0; i < 4; i++ {
		resp, _ = app.Test(httptest.NewRequest(http.MethodGet, "/protected", nil), -1)
		resp.Body.Close()
	}

	if got := repo.findCalls.Load(); got != 1 {
		t.Fatalf("expected exactly 1 DB lookup with cache, got %d", got)
	}
}

func TestBan_CacheExpiryRefetchesFromDB(t *testing.T) {
	id := uuid.New()
	repo := newFakeUserRepo()
	repo.users[id] = &entity.User{User: &ent.User{ID: id, IsBanned: false}}

	// 시간 제어 가능한 캐시. 매우 짧은 TTL.
	cache := NewBanCache(50 * time.Millisecond)
	app := setupBanApp(t, id, repo, cache)

	// 1차 — 캐시 적재 (false)
	resp, _ := app.Test(httptest.NewRequest(http.MethodGet, "/protected", nil), -1)
	resp.Body.Close()

	// DB 에서 ban 토글 (운영자가 UPDATE 했다는 시나리오)
	repo.users[id].IsBanned = true

	// 캐시 만료 전: 여전히 통과
	resp, _ = app.Test(httptest.NewRequest(http.MethodGet, "/protected", nil), -1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("within TTL banned change should not reflect yet; got %d", resp.StatusCode)
	}

	time.Sleep(80 * time.Millisecond)

	// 캐시 만료 후: DB 재조회 → 403
	resp, _ = app.Test(httptest.NewRequest(http.MethodGet, "/protected", nil), -1)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("after TTL ban should take effect; got %d", resp.StatusCode)
	}
}

func TestBan_InvalidateForcesRefetch(t *testing.T) {
	id := uuid.New()
	repo := newFakeUserRepo()
	repo.users[id] = &entity.User{User: &ent.User{ID: id, IsBanned: false}}
	cache := NewBanCache(time.Hour)
	app := setupBanApp(t, id, repo, cache)

	// 1차 — 캐시 적재
	resp, _ := app.Test(httptest.NewRequest(http.MethodGet, "/protected", nil), -1)
	resp.Body.Close()

	// DB ban + 캐시 즉시 무효화 (향후 ban toggle 핸들러가 호출할 경로)
	repo.users[id].IsBanned = true
	cache.Invalidate(id)

	// 다음 요청에서 DB 재조회 → 403
	resp, _ = app.Test(httptest.NewRequest(http.MethodGet, "/protected", nil), -1)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("Invalidate should force refetch; got %d", resp.StatusCode)
	}
}

func TestBan_DBErrorFailsOpen(t *testing.T) {
	// 사용자 레코드가 repo 에 없으면 (DB miss) fail-open: 통과.
	// 일시적 장애로 매 요청을 차단하는 일을 피한다.
	id := uuid.New()
	repo := newFakeUserRepo() // 비어있음
	app := setupBanApp(t, id, repo, NewBanCache(time.Minute))

	resp, _ := app.Test(httptest.NewRequest(http.MethodGet, "/protected", nil), -1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("DB miss should fail-open; got %d", resp.StatusCode)
	}
}

func readResp(t *testing.T, app *fiber.App, req *http.Request) (*http.Response, []byte) {
	t.Helper()
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, b
}
