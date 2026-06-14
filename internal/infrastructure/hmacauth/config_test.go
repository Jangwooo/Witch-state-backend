package hmacauth

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

func TestLoadConfigDefaults(t *testing.T) {
	viper.Reset()
	cfg := LoadConfig()
	if cfg.Mode != ModeOff {
		t.Fatalf("default mode should be off, got %s", cfg.Mode)
	}
	if cfg.IsIssuingSecret() {
		t.Fatal("off mode should not issue secret")
	}
	if cfg.EnforcePct != 0 {
		t.Fatalf("default EnforcePct should be 0, got %d", cfg.EnforcePct)
	}
	enf, reason := cfg.ShouldEnforce("00000000-0000-0000-0000-000000000001", "steam", "abc")
	if enf || reason != EnforceReasonModeOff {
		t.Fatalf("off mode should not enforce; got (%v, %s)", enf, reason)
	}
}

func TestLoadConfigShadow(t *testing.T) {
	viper.Reset()
	viper.Set("HMAC_MODE", "shadow")
	cfg := LoadConfig()
	if cfg.Mode != ModeShadow {
		t.Fatalf("expected shadow, got %s", cfg.Mode)
	}
	if !cfg.IsIssuingSecret() {
		t.Fatal("shadow should issue secret")
	}
	enf, reason := cfg.ShouldEnforce("00000000-0000-0000-0000-000000000001", "steam", "abc")
	if enf || reason != EnforceReasonModeOff {
		t.Fatalf("shadow should never enforce; got (%v, %s)", enf, reason)
	}
}

func TestLoadConfigEnforceWhitelist(t *testing.T) {
	viper.Reset()
	viper.Set("HMAC_MODE", "enforce")
	viper.Set("HMAC_ENFORCE_PLATFORM_IDS", "steam:7656,STOVE:99 , ")
	cfg := LoadConfig()
	if cfg.Mode != ModeEnforce {
		t.Fatalf("expected enforce, got %s", cfg.Mode)
	}
	if !cfg.IsIssuingSecret() {
		t.Fatal("enforce should issue secret")
	}
	enf, reason := cfg.ShouldEnforce("u1", "steam", "7656")
	if !enf || reason != EnforceReasonWhitelist {
		t.Fatalf("whitelisted steam id should enforce; got (%v, %s)", enf, reason)
	}
	enf, reason = cfg.ShouldEnforce("u1", "Stove", "99")
	if !enf || reason != EnforceReasonWhitelist {
		t.Fatalf("case-insensitive stove match should enforce; got (%v, %s)", enf, reason)
	}
	// PCT 미설정(=0) 이고 화이트리스트 밖이면 enforce 안 함
	enf, reason = cfg.ShouldEnforce("u1", "steam", "0000")
	if enf || reason != EnforceReasonPctSkipped {
		t.Fatalf("non-whitelisted id with PCT=0 should not enforce; got (%v, %s)", enf, reason)
	}
}

func TestEnforceWithEmptyWhitelistAndZeroPct(t *testing.T) {
	viper.Reset()
	viper.Set("HMAC_MODE", "enforce")
	cfg := LoadConfig()
	enf, reason := cfg.ShouldEnforce("00000000-0000-0000-0000-000000000001", "steam", "anyone")
	if enf || reason != EnforceReasonPctSkipped {
		t.Fatalf("empty whitelist + PCT=0 should never enforce; got (%v, %s)", enf, reason)
	}
}

func TestEnforcePct100AppliesToAll(t *testing.T) {
	viper.Reset()
	viper.Set("HMAC_MODE", "enforce")
	viper.Set("HMAC_ENFORCE_PCT", "100")
	cfg := LoadConfig()
	for i := 0; i < 50; i++ {
		uid := uuid.New().String()
		enf, reason := cfg.ShouldEnforce(uid, "steam", fmt.Sprintf("p%d", i))
		if !enf || reason != EnforceReasonPctBucket {
			t.Fatalf("PCT=100 should enforce all; uid=%s got (%v, %s)", uid, enf, reason)
		}
	}
}

func TestEnforcePctClamping(t *testing.T) {
	viper.Reset()
	viper.Set("HMAC_MODE", "enforce")
	viper.Set("HMAC_ENFORCE_PCT", "9999")
	cfg := LoadConfig()
	if cfg.EnforcePct != 100 {
		t.Fatalf("9999 should clamp to 100, got %d", cfg.EnforcePct)
	}

	viper.Set("HMAC_ENFORCE_PCT", "-5")
	cfg = LoadConfig()
	if cfg.EnforcePct != 0 {
		t.Fatalf("-5 should clamp to 0, got %d", cfg.EnforcePct)
	}
}

// 결정적: 같은 user_id 는 PCT 가 어떻든 같은 bucket 으로 분류된다.
// 따라서 PCT 를 올리면 enforce 집합은 monotonic 하게 커진다.
func TestEnforcePctMonotonicity(t *testing.T) {
	// 결정적 임의 user_id 1000개 (해시 결과로만 판정되므로 시드 불필요)
	ids := make([]string, 1000)
	for i := range ids {
		ids[i] = uuid.NewSHA1(uuid.NameSpaceDNS, []byte(fmt.Sprintf("user-%d", i))).String()
	}

	enforcedAt := func(pct int) map[string]struct{} {
		viper.Reset()
		viper.Set("HMAC_MODE", "enforce")
		viper.Set("HMAC_ENFORCE_PCT", fmt.Sprintf("%d", pct))
		cfg := LoadConfig()
		set := make(map[string]struct{})
		for _, id := range ids {
			enf, _ := cfg.ShouldEnforce(id, "steam", "")
			if enf {
				set[id] = struct{}{}
			}
		}
		return set
	}

	at10 := enforcedAt(10)
	at50 := enforcedAt(50)
	at100 := enforcedAt(100)

	for id := range at10 {
		if _, ok := at50[id]; !ok {
			t.Fatalf("monotonicity violated: id %s enforced at PCT=10 but not at PCT=50", id)
		}
	}
	for id := range at50 {
		if _, ok := at100[id]; !ok {
			t.Fatalf("monotonicity violated: id %s enforced at PCT=50 but not at PCT=100", id)
		}
	}
}

// 균등성: 1만 개 UUID 를 100개 bucket 으로 흩으면 bucket 당 평균 100 ±오차 수준.
// 카이제곱 정밀 검정 대신 "어떤 bucket 도 평균의 ±50% 안" 정도로 느슨하게 둔다.
func TestUserIDBucketDistribution(t *testing.T) {
	const n = 10000
	buckets := make([]int, 100)
	for i := 0; i < n; i++ {
		uid := uuid.NewSHA1(uuid.NameSpaceDNS, []byte(fmt.Sprintf("dist-%d", i))).String()
		buckets[userIDBucket(uid)]++
	}
	mean := float64(n) / 100.0
	lo, hi := mean*0.5, mean*1.5
	for i, c := range buckets {
		if float64(c) < lo || float64(c) > hi {
			t.Fatalf("bucket %d count=%d outside [%v, %v] — distribution skewed", i, c, lo, hi)
		}
	}
}

func TestUserIDBucketDeterministic(t *testing.T) {
	uid := "550e8400-e29b-41d4-a716-446655440000"
	if userIDBucket(uid) != userIDBucket(uid) {
		t.Fatal("same input must produce same bucket")
	}
	// 케이스/공백 정규화
	if userIDBucket("550E8400-E29B-41D4-A716-446655440000") != userIDBucket(uid) {
		t.Fatal("upper/lower case UUID should produce same bucket")
	}
	if userIDBucket("  "+uid+"  ") != userIDBucket(uid) {
		t.Fatal("whitespace-padded UUID should produce same bucket")
	}
}

func TestEnforceNoUserIDFallback(t *testing.T) {
	viper.Reset()
	viper.Set("HMAC_MODE", "enforce")
	viper.Set("HMAC_ENFORCE_PCT", "50")
	cfg := LoadConfig()
	enf, reason := cfg.ShouldEnforce("", "steam", "abc")
	if enf || reason != EnforceReasonNoUserID {
		t.Fatalf("missing userID with PCT=50 should fail-safe to (false, no_user_id); got (%v, %s)", enf, reason)
	}
}

// 화이트리스트는 PCT 와 병행 — userID 와 무관하게 우선.
func TestWhitelistOverridesPct(t *testing.T) {
	viper.Reset()
	viper.Set("HMAC_MODE", "enforce")
	viper.Set("HMAC_ENFORCE_PCT", "0")
	viper.Set("HMAC_ENFORCE_PLATFORM_IDS", "steam:tester")
	cfg := LoadConfig()
	enf, reason := cfg.ShouldEnforce(uuid.New().String(), "steam", "tester")
	if !enf || reason != EnforceReasonWhitelist {
		t.Fatalf("whitelist should override PCT=0; got (%v, %s)", enf, reason)
	}
}
