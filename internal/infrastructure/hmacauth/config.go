package hmacauth

import (
	"crypto/sha256"
	"encoding/binary"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

// Mode 는 HMAC 검증 모드입니다.
//   - ModeOff: 미들웨어 no-op. signin 응답에 hmac_secret 도 포함하지 않음. 100% 기존 동작.
//   - ModeShadow: hmac_secret 발급. 헤더가 오면 검증해 로그만 남김. 실패해도 200 통과.
//   - ModeEnforce: 검증 실패 시 401. 단 테스터 화이트리스트에 한해 적용.
//     화이트리스트 밖 유저는 헤더 없으면 통과(= shadow 처럼 동작).
type Mode string

const (
	ModeOff     Mode = "off"
	ModeShadow  Mode = "shadow"
	ModeEnforce Mode = "enforce"
)

// Config 는 환경변수 기반 HMAC 동작 설정입니다.
type Config struct {
	Mode Mode

	// EnforcePlatformIDs 는 enforce 모드에서 검증 실패 시 401 을 반환할 대상 화이트리스트입니다.
	// 키 포맷: "platform_type:platform_user_id" (예: "steam:76561198000000000", "stove:1234567")
	// 매칭은 대소문자 무시. 화이트리스트는 EnforcePct 와 병행 동작 — 화이트리스트가 우선.
	EnforcePlatformIDs map[string]struct{}

	// EnforcePct 는 화이트리스트 외 일반 유저 중 몇 % 를 enforce 대상으로 삼을지 결정합니다.
	// 0 = 일반 유저 미적용 (화이트리스트만), 100 = 전체 enforce. 기본 0.
	EnforcePct int
}

// LoadConfig 는 viper 에서 HMAC_MODE / HMAC_ENFORCE_PLATFORM_IDS / HMAC_ENFORCE_PCT 를 읽어 Config 를 만듭니다.
//   - HMAC_MODE: off | shadow | enforce. 미설정/오타 시 off.
//   - HMAC_ENFORCE_PLATFORM_IDS: "steam:abc,stove:123" 형식 콤마 구분 리스트.
//   - HMAC_ENFORCE_PCT: 0..100. 범위 밖이면 0/100 으로 클램프. enforce 모드에서만 의미.
func LoadConfig() *Config {
	mode := parseMode(viper.GetString("HMAC_MODE"))
	whitelist := parsePlatformIDs(viper.GetString("HMAC_ENFORCE_PLATFORM_IDS"))
	pct := clampPct(viper.GetInt("HMAC_ENFORCE_PCT"))
	return &Config{
		Mode:               mode,
		EnforcePlatformIDs: whitelist,
		EnforcePct:         pct,
	}
}

func clampPct(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func parseMode(raw string) Mode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "shadow":
		return ModeShadow
	case "enforce":
		return ModeEnforce
	default:
		return ModeOff
	}
}

func parsePlatformIDs(raw string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, token := range strings.Split(raw, ",") {
		token = strings.ToLower(strings.TrimSpace(token))
		if token == "" {
			continue
		}
		result[token] = struct{}{}
	}
	return result
}

// IsIssuingSecret 은 signin 응답에 hmac_secret 을 포함시켜야 하는지 반환합니다.
// off 모드에서는 발급조차 하지 않아 클라가 헤더를 부착하지 않게 유도합니다.
func (c *Config) IsIssuingSecret() bool {
	return c.Mode == ModeShadow || c.Mode == ModeEnforce
}

// EnforceReason 은 should_enforce 판정 사유를 운영/디버그 로그용으로 노출합니다.
type EnforceReason string

const (
	EnforceReasonModeOff    EnforceReason = "mode_off"     // enforce 모드가 아니거나 off/shadow
	EnforceReasonWhitelist  EnforceReason = "whitelist"    // platform_user_id 화이트리스트 매칭
	EnforceReasonPctBucket  EnforceReason = "pct_bucket"   // 결정적 해시 bucket < EnforcePct
	EnforceReasonPctSkipped EnforceReason = "pct_skipped"  // bucket >= EnforcePct (or PCT=0)
	EnforceReasonNoUserID   EnforceReason = "no_user_id"   // user.id 미존재 — 안전상 enforce 안 함
)

// ShouldEnforce 는 enforce 모드 진입 + 화이트리스트 매칭 + PCT bucket 판정 결과를 반환합니다.
// 결정적 — 같은 userID 는 항상 같은 결과.
//   - userID: users 테이블 PK (UUID). PCT bucket 판정의 안정적 식별자.
//   - platformType / platformUserID: 화이트리스트 매칭 키.
//
// 우선순위:
//  1. Mode != enforce → (false, mode_off)
//  2. 화이트리스트 매칭 → (true, whitelist)
//  3. EnforcePct >= 100 → (true, pct_bucket)
//  4. EnforcePct <= 0 → (false, pct_skipped)
//  5. SHA-256(userID) 앞 4바이트 → uint32 → %100. PCT 미만이면 (true, pct_bucket), 이상이면 (false, pct_skipped)
//  6. userID 빈 문자열인데 위 1~4 가 모두 결정 못 함 → (false, no_user_id) — 안전 폴백
func (c *Config) ShouldEnforce(userID, platformType, platformUserID string) (bool, EnforceReason) {
	if c.Mode != ModeEnforce {
		return false, EnforceReasonModeOff
	}

	if matchesWhitelist(c.EnforcePlatformIDs, platformType, platformUserID) {
		return true, EnforceReasonWhitelist
	}

	if c.EnforcePct >= 100 {
		return true, EnforceReasonPctBucket
	}
	if c.EnforcePct <= 0 {
		return false, EnforceReasonPctSkipped
	}

	if strings.TrimSpace(userID) == "" {
		// PCT 가 0~100 사이인데 PK 가 없으면 안정적 bucket 을 계산할 수 없음.
		// 안전상 enforce 미적용(= shadow 처럼 통과). 호출자가 로그로 추적.
		return false, EnforceReasonNoUserID
	}

	bucket := userIDBucket(userID)
	if int(bucket) < c.EnforcePct {
		return true, EnforceReasonPctBucket
	}
	return false, EnforceReasonPctSkipped
}

// userIDBucket 은 userID 를 SHA-256 해시한 뒤 앞 4바이트(uint32) % 100 으로 0..99 bucket 을 만듭니다.
// 같은 userID 는 항상 같은 bucket 을 반환합니다. UUID 의 정규화(소문자, 공백 제거) 후 바이트 그대로 해시 입력.
func userIDBucket(userID string) uint32 {
	normalized := strings.ToLower(strings.TrimSpace(userID))
	// UUID 문자열로 들어오면 표준 포맷으로 통일해 같은 키가 다른 포맷일 때도 같은 bucket 이 되게 한다.
	if parsed, err := uuid.Parse(normalized); err == nil {
		normalized = parsed.String()
	}
	sum := sha256.Sum256([]byte(normalized))
	return binary.BigEndian.Uint32(sum[:4]) % 100
}

func matchesWhitelist(set map[string]struct{}, platformType, platformUserID string) bool {
	if len(set) == 0 {
		return false
	}
	key := strings.ToLower(strings.TrimSpace(platformType) + ":" + strings.TrimSpace(platformUserID))
	_, ok := set[key]
	return ok
}
