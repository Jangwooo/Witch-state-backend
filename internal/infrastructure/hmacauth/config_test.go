package hmacauth

import (
	"testing"

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
	if cfg.ShouldEnforce("steam", "abc") {
		t.Fatal("off mode should not enforce")
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
	if cfg.ShouldEnforce("steam", "abc") {
		t.Fatal("shadow should never enforce")
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
	if !cfg.ShouldEnforce("steam", "7656") {
		t.Fatal("whitelisted steam id should enforce")
	}
	if !cfg.ShouldEnforce("Stove", "99") {
		t.Fatal("case-insensitive stove match should enforce")
	}
	if cfg.ShouldEnforce("steam", "0000") {
		t.Fatal("non-whitelisted id should not enforce")
	}
}

func TestEnforceWithEmptyWhitelist(t *testing.T) {
	viper.Reset()
	viper.Set("HMAC_MODE", "enforce")
	cfg := LoadConfig()
	if cfg.ShouldEnforce("steam", "anyone") {
		t.Fatal("empty whitelist should never enforce — fail-safe")
	}
}
