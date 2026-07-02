package middleware

import (
	"context"
	"crypto/subtle"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/domain/repository"
)

// ExhibitionHeaderKey 는 전시 빌드가 부착하는 사전 공유 키 헤더 이름이다 (exhibition-logging Q4).
const ExhibitionHeaderKey = "X-Exhibition-Key"

// exhibitionLocalsKey 는 전시 요청 여부를 담는 c.Locals 키. 후속 미들웨어가 우회 판정에 쓴다.
const exhibitionLocalsKey = "is_exhibition"

// ExhibitionGateConfig 는 전시 게이트 미들웨어의 의존성 묶음이다.
type ExhibitionGateConfig struct {
	// Key 는 사전 공유 키 (env EXHIBITION_KEY). 빈 문자열이면 게이트 비활성 — 어떤 요청도 전시로 통과시키지 않음.
	Key string
	// User 는 전시 전용 고정 계정. 부팅 시 EXHIBITION_USER_ID 로 조회해 주입한다.
	// nil 이면 게이트 비활성 (오설정 시 안전하게 기능 off — 전시 요청은 일반 인증 체인으로 폴백해 401).
	User *entity.User
}

// ExhibitionGate 는 record 라우트 최전단 미들웨어다 (exhibition-logging Q4, 제안 A).
//
// 유효한 X-Exhibition-Key 가 있고 게이트가 활성(Key + User 모두 설정)이면:
//   - 전시 고정 계정을 c.Locals("user") 에 주입
//   - c.Locals("is_exhibition")=true 세팅
// 후속 미들웨어(AuthMiddleware / BanCheckMiddleware / HMACMiddleware)는 각자 IsExhibition(c)
// 를 확인해 전시 요청이면 즉시 통과한다. 따라서 전시=세션 없음/HMAC 미부착이어도 401 이 나지 않는다.
//
// 전시가 아니면(헤더 없음/불일치/게이트 비활성) 아무 것도 세팅하지 않고 그대로 c.Next() —
// 비전시 요청 경로는 전혀 바뀌지 않는다 (baseline 보존).
func ExhibitionGate(cfg ExhibitionGateConfig) fiber.Handler {
	enabled := cfg.Key != "" && cfg.User != nil && cfg.User.User != nil

	return func(c *fiber.Ctx) error {
		if enabled && exhibitionKeyMatches(c.Get(ExhibitionHeaderKey), cfg.Key) {
			c.Locals("user", cfg.User)
			c.Locals(exhibitionLocalsKey, true)
		}
		return c.Next()
	}
}

// IsExhibition 은 ExhibitionGate 가 전시 요청으로 판정해 표식을 남겼는지 반환한다.
// 인증 계열 미들웨어가 전시 요청을 우회시킬 때 쓴다.
func IsExhibition(c *fiber.Ctx) bool {
	v, _ := c.Locals(exhibitionLocalsKey).(bool)
	return v
}

// exhibitionKeyMatches 는 상수 시간 비교로 키 일치를 확인한다 (타이밍 사이드채널 방지).
func exhibitionKeyMatches(got, want string) bool {
	if got == "" || want == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

// LoadExhibitionGate 는 env 에서 전시 게이트 설정을 로드한다.
//   - EXHIBITION_KEY: 사전 공유 키. 미설정이면 게이트 비활성.
//   - EXHIBITION_USER_ID: 전시 전용 고정 계정 UUID. 미설정/미존재면 게이트 비활성.
// 둘 다 유효할 때만 전시 요청을 통과시킨다 (오설정 시 안전하게 기능 off — 기존 동작 불변).
func LoadExhibitionGate(ctx context.Context, userRepo repository.UserRepository) ExhibitionGateConfig {
	key := viper.GetString("EXHIBITION_KEY")
	userID := viper.GetString("EXHIBITION_USER_ID")
	return ExhibitionGateConfig{
		Key:  key,
		User: LoadExhibitionUser(ctx, userRepo, userID),
	}
}

// LoadExhibitionUser 는 EXHIBITION_USER_ID(UUID)로 전시 고정 계정을 조회한다.
// 미설정/파싱 실패/미존재면 nil 을 반환해 게이트가 비활성되도록 한다 (오설정 안전).
func LoadExhibitionUser(ctx context.Context, userRepo repository.UserRepository, rawUserID string) *entity.User {
	if rawUserID == "" || userRepo == nil {
		return nil
	}
	id, err := uuid.Parse(rawUserID)
	if err != nil {
		return nil
	}
	usr, err := userRepo.FindByID(ctx, id)
	if err != nil || usr == nil || usr.User == nil {
		return nil
	}
	return usr
}
