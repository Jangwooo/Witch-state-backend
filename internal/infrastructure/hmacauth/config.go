package hmacauth

import (
	"strings"

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
	// 매칭은 대소문자 무시.
	EnforcePlatformIDs map[string]struct{}
}

// LoadConfig 는 viper 에서 HMAC_MODE / HMAC_ENFORCE_PLATFORM_IDS 를 읽어 Config 를 만듭니다.
//   - HMAC_MODE: off | shadow | enforce. 미설정/오타 시 off.
//   - HMAC_ENFORCE_PLATFORM_IDS: "steam:abc,stove:123" 형식 콤마 구분 리스트.
func LoadConfig() *Config {
	mode := parseMode(viper.GetString("HMAC_MODE"))
	whitelist := parsePlatformIDs(viper.GetString("HMAC_ENFORCE_PLATFORM_IDS"))
	return &Config{
		Mode:               mode,
		EnforcePlatformIDs: whitelist,
	}
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

// ShouldEnforce 는 enforce 모드 + 화이트리스트 매칭 시에만 true.
// 즉, 검증 실패 시 401 을 반환할지 결정합니다.
func (c *Config) ShouldEnforce(platformType, platformUserID string) bool {
	if c.Mode != ModeEnforce {
		return false
	}
	if len(c.EnforcePlatformIDs) == 0 {
		return false
	}
	key := strings.ToLower(strings.TrimSpace(platformType) + ":" + strings.TrimSpace(platformUserID))
	_, ok := c.EnforcePlatformIDs[key]
	return ok
}
