package middleware

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/domain/repository"
)

// BanCheckTTL 은 users.is_banned 캐시 TTL 입니다.
// 명세 2.2 절: 운영자가 DB 직접 UPDATE 로 토글하므로 즉시 무효화 경로가 없습니다.
// 60 초면 "밴 후 한 클릭 내 차단" 요건을 사실상 만족하면서 매 요청 DB 조회 부하를 막습니다.
const BanCheckTTL = 60 * time.Second

// BanCache 는 users.is_banned 값을 TTL 캐시합니다. 동시 호출 안전.
type BanCache struct {
	mu      sync.RWMutex
	entries map[uuid.UUID]banEntry
	ttl     time.Duration
	nowFn   func() time.Time
}

type banEntry struct {
	banned    bool
	expiresAt time.Time
}

// NewBanCache 는 BanCache 를 생성합니다. ttl 이 0 이면 BanCheckTTL.
func NewBanCache(ttl time.Duration) *BanCache {
	if ttl <= 0 {
		ttl = BanCheckTTL
	}
	return &BanCache{
		entries: make(map[uuid.UUID]banEntry),
		ttl:     ttl,
		nowFn:   time.Now,
	}
}

// lookup 은 캐시 조회. (banned, hit). hit=false 면 DB 조회 필요.
func (c *BanCache) lookup(id uuid.UUID) (bool, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[id]
	if !ok {
		return false, false
	}
	if c.nowFn().After(e.expiresAt) {
		return false, false
	}
	return e.banned, true
}

func (c *BanCache) set(id uuid.UUID, banned bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[id] = banEntry{banned: banned, expiresAt: c.nowFn().Add(c.ttl)}
}

// Invalidate 는 캐시에서 항목을 제거합니다. 향후 ban toggle 핸들러가 추가되면 호출 지점.
func (c *BanCache) Invalidate(id uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, id)
}

// BanCheckMiddleware 는 c.Locals("user") 에 세팅된 유저의 is_banned 를 캐시→DB 순으로 확인하고,
// 차단된 사용자는 403 + account_banned 로 응답합니다. AuthMiddleware 직후에 등록되어야 합니다.
//
// userRepo 는 FindByID 로 신선한 users 레코드를 조회합니다 (캐시 미스 시).
// cache 는 nil 도 허용 — 매 요청 DB 조회로 동작합니다.
func BanCheckMiddleware(userRepo repository.UserRepository, cache *BanCache) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 전시(EXHIBITION) 요청은 밴 체크 대상이 아니다 (고정 계정, exhibition-logging Q4).
		if IsExhibition(c) {
			return c.Next()
		}

		usr, _ := c.Locals("user").(*entity.User)
		if usr == nil || usr.User == nil {
			// AuthMiddleware 가 이미 차단했어야 하지만, 방어적으로 통과.
			return c.Next()
		}

		// 1) 세션 페이로드에 박혀있던 IsBanned 값은 신뢰 안 함 (세션 생성 시점 기준).
		//    캐시 → DB 순으로 신선한 값 확인.
		id := usr.ID

		if cache != nil {
			if banned, hit := cache.lookup(id); hit {
				if banned {
					return respondBanned(c)
				}
				return c.Next()
			}
		}

		// 2) 캐시 미스: DB 조회.
		fresh, err := userRepo.FindByID(c.Context(), id)
		if err != nil {
			// 조회 실패는 보안 측면에서 fail-closed 가 맞지만, 운영 가용성 측면에서 fail-open.
			// 캐시 갱신 안 하고 일단 통과시켜 다음 요청에서 재시도.
			return c.Next()
		}
		banned := false
		if fresh != nil && fresh.User != nil {
			banned = fresh.IsBanned
		}
		if cache != nil {
			cache.set(id, banned)
		}
		if banned {
			return respondBanned(c)
		}
		return c.Next()
	}
}

func respondBanned(c *fiber.Ctx) error {
	return c.Status(fiber.StatusForbidden).JSON(entity.ErrorResponse{
		Message: "차단된 계정입니다.",
		Error:   "account_banned",
		Details: fiber.Map{"reason": "is_banned"},
	})
}

