package middleware

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/infrastructure/hmacauth"
	"github.com/witchs-lounge_backend/internal/infrastructure/logging"
)

// NonceStore 는 nonce 리플레이 차단용 키-값 캐시 인터페이스입니다.
// 운영에선 redis.Client 로 SET NX 패턴을 사용하고, 테스트는 인메모리 fake 로 대체합니다.
type NonceStore interface {
	// SetNX 는 이미 키가 있으면 false (= 리플레이), 새로 세팅하면 true 를 반환합니다.
	// Redis 장애 등 에러는 error 로 반환합니다.
	SetNX(ctx context.Context, key string, ttl time.Duration) (bool, error)
}

// RedisNonceStore 는 go-redis 기반 NonceStore 구현입니다.
type RedisNonceStore struct {
	Client *redis.Client
}

func (r *RedisNonceStore) SetNX(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if r == nil || r.Client == nil {
		return true, nil // Redis 미설정 시 검증 미수행
	}
	return r.Client.SetNX(ctx, key, "1", ttl).Result()
}

// 명세 4절의 검증 윈도우/캐시 TTL.
const (
	timestampSkewSeconds = 300              // ±5분
	nonceTTL             = 600 * time.Second // 10분
)

// HMACVerifierConfig 는 HMACMiddleware 의 의존성 묶음입니다.
type HMACVerifierConfig struct {
	Config      *hmacauth.Config
	NonceStore  NonceStore
	ErrorLogger *logging.ErrorLogger
	// NowFunc 은 테스트용 시각 주입. nil 이면 time.Now 사용.
	NowFunc func() time.Time
}

// HMACMiddleware 는 명세 4절의 5단계 검증을 수행합니다.
//
// 동작:
//   - HMAC_MODE=off: 어떤 검증도 하지 않고 즉시 통과 (no-op).
//   - HMAC_MODE=shadow: 검증해 실패 사유를 구조화 로그로 남김. 200 통과.
//     PCT/PLATFORM_IDS 는 shadow 에 영향을 주지 않음.
//   - HMAC_MODE=enforce: 화이트리스트 또는 결정적 PCT bucket 에 들어가는 유저는 검증 실패 시 401.
//     그 외 유저는 shadow 처럼 통과(= 일반 사용자 무영향).
//
// 전제: AuthMiddleware 가 먼저 실행되어 c.Locals("user"), c.Locals("sessionID") 가 세팅돼 있어야 함.
func HMACMiddleware(cfg HMACVerifierConfig) fiber.Handler {
	nowFn := cfg.NowFunc
	if nowFn == nil {
		nowFn = time.Now
	}

	return func(c *fiber.Ctx) error {
		// 전시(EXHIBITION) 요청은 세션이 없어 HMAC 시크릿도 없다 (exhibition-logging Q4).
		// 클라도 X-Signature 를 부착하지 않으므로 검증을 건너뛴다.
		if IsExhibition(c) {
			return c.Next()
		}

		hcfg := cfg.Config
		if hcfg == nil || hcfg.Mode == hmacauth.ModeOff {
			return c.Next()
		}

		usr, _ := c.Locals("user").(*entity.User)
		sessionID, _ := c.Locals("sessionID").(string)
		// AuthMiddleware 가 같은 트랜잭션에서 이미 조회해 Locals 에 넣어둠.
		hmacSecret, _ := c.Locals("hmacSecret").(string)

		// 검증 시도 → 실패 사유 정리. nil 이면 통과.
		failReason := verifyRequest(c, hmacSecret, cfg.NonceStore, nowFn)

		if failReason == nil {
			c.Locals("hmac_verified", true)
			return c.Next()
		}

		enforced, enforceReason := shouldEnforce(hcfg, usr)

		// 실패 사유 + enforce 판정 로그 (시크릿 절대 노출 금지)
		logHMACFailure(cfg.ErrorLogger, c, sessionID, usr, hmacSecret, failReason, enforced, enforceReason)

		// enforce + 대상 매칭 시에만 401.
		if enforced {
			return c.Status(fiber.StatusUnauthorized).JSON(entity.ErrorResponse{
				Message: "요청 서명 검증 실패",
				Error:   "hmac_verification_failed",
				Details: fiber.Map{"reason": failReason.Error()},
			})
		}

		c.Locals("hmac_verified", false)
		return c.Next()
	}
}

// shouldEnforce 는 사용자가 enforce 대상인지 판정합니다.
// 첫 반환값은 401 처리 여부, 두 번째는 판정 사유(로그용).
func shouldEnforce(hcfg *hmacauth.Config, usr *entity.User) (bool, hmacauth.EnforceReason) {
	if usr == nil || usr.User == nil {
		return false, hmacauth.EnforceReasonNoUserID
	}
	return hcfg.ShouldEnforce(usr.ID.String(), string(usr.PlatformType), usr.PlatformUserID)
}

// verifyRequest 는 명세 4절 1~5 단계를 순서대로 검사합니다.
// 통과 시 nil, 실패 시 사유 error 를 반환합니다.
func verifyRequest(c *fiber.Ctx, hmacSecret string, nonces NonceStore, nowFn func() time.Time) error {
	sig := c.Get("X-Signature")
	nonce := c.Get("X-Nonce")
	tsStr := c.Get("X-Timestamp")

	// 헤더 부재 — 구버전 클라일 수 있음. "missing"으로 명시.
	if sig == "" || nonce == "" || tsStr == "" {
		return errors.New("missing_headers")
	}

	if !hmacauth.IsValidNonceFormat(nonce) {
		return errors.New("invalid_nonce_format")
	}

	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return errors.New("invalid_timestamp_format")
	}

	now := nowFn().Unix()
	if diff := now - ts; diff > timestampSkewSeconds || diff < -timestampSkewSeconds {
		return errors.New("timestamp_skew")
	}

	// nonce 리플레이 차단.
	if nonces != nil {
		set, rerr := nonces.SetNX(c.Context(), "hmac_nonce:"+nonce, nonceTTL)
		if rerr != nil {
			// 캐시 장애 시 검증을 막아 실패-오픈으로 가지 않도록 명시적으로 실패로 처리.
			return errors.New("nonce_store_error")
		}
		if !set {
			return errors.New("nonce_replay")
		}
	}

	// grace period: 구버전 세션(hmac_secret 없음) 은 검증 불가 → 통과.
	// 단, 클라가 헤더는 부착했지만 서버 세션에 시크릿이 없다 = 비정상(다른 세션의 헤더?).
	// shadow 단계에서 발견하기 위해 명시적 실패로 남긴다. enforce 라도 화이트리스트가 아니면 통과한다.
	if hmacSecret == "" {
		return errors.New("session_missing_secret")
	}

	canonical := hmacauth.BuildCanonical(
		c.Method(),
		c.OriginalURL(),
		nonce,
		tsStr,
		hmacauth.BodySHA256Hex(c.Body()),
	)
	expected := hmacauth.Sign(hmacSecret, canonical)
	if !hmacauth.VerifySignature(expected, strings.ToLower(sig)) {
		return errors.New("signature_mismatch")
	}

	return nil
}

// logHMACFailure 는 보안 로그를 남깁니다.
// hmac_secret 은 절대 노출하지 않고 앞 4자만 마스킹된 형태로만 노출합니다.
func logHMACFailure(l *logging.ErrorLogger, c *fiber.Ctx, sessionID string, usr *entity.User, secret string, reason error, enforced bool, enforceReason hmacauth.EnforceReason) {
	if l == nil {
		return
	}
	// 도메인 entity 에 의존하지 않는 보조 식별자.
	platform, platformUser, userIDShort := "", "", ""
	if usr != nil && usr.User != nil {
		platform = string(usr.PlatformType)
		platformUser = usr.PlatformUserID
		userIDShort = shortID(usr.ID.String())
	}

	// ErrorLogger 의 sanitize 로직이 헤더의 sensitive 키를 자동으로 마스킹합니다.
	// signin 응답에 포함되는 hmac_secret 도 이미 sanitizeJSONValue 로 마스킹됩니다.
	// 우리는 추가로 마스킹된 hint 만 응답 details 에 남깁니다.
	hint := hmacauth.MaskSecret(secret)
	l.LogHMACVerification(c, reason.Error(), map[string]interface{}{
		"session_id_short":  shortID(sessionID),
		"user_id_short":     userIDShort,
		"platform":          platform,
		"platform_user_id":  platformUser,
		"secret_hint":       hint,
		"x_nonce":           c.Get("X-Nonce"),
		"x_timestamp":       c.Get("X-Timestamp"),
		"x_signature_short": shortSig(c.Get("X-Signature")),
		"method":            c.Method(),
		"path_and_query":    c.OriginalURL(),
		"server_unix_ts":    time.Now().Unix(),
		"would_enforce":     enforced,
		"enforce_reason":    string(enforceReason),
	})
}

// shortID, shortSig 는 디버그용 짧은 식별자만 노출합니다.
func shortID(s string) string {
	if len(s) <= 8 {
		return s
	}
	return s[:8] + "…"
}

func shortSig(s string) string {
	if len(s) <= 12 {
		return s
	}
	return s[:8] + "…" + s[len(s)-4:]
}

