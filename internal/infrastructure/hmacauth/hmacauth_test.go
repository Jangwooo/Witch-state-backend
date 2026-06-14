package hmacauth

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestEmptyBodySHA256Hex(t *testing.T) {
	sum := sha256.Sum256(nil)
	if got := hex.EncodeToString(sum[:]); got != EmptyBodySHA256Hex {
		t.Fatalf("EmptyBodySHA256Hex constant mismatch: got %s want %s", got, EmptyBodySHA256Hex)
	}
}

func TestBuildCanonical(t *testing.T) {
	canonical := BuildCanonical(
		"get",
		"/api/v1/records?music_id=test",
		"9f1a2c3b4d5e6f708192a3b4c5d6e7f8",
		"1750000000",
		EmptyBodySHA256Hex,
	)
	want := "GET\n/api/v1/records?music_id=test\n9f1a2c3b4d5e6f708192a3b4c5d6e7f8\n1750000000\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if canonical != want {
		t.Fatalf("canonical mismatch\n got: %q\nwant: %q", canonical, want)
	}
}

func TestSignFixture(t *testing.T) {
	canonical := BuildCanonical(
		"GET",
		"/api/v1/records?music_id=test",
		"9f1a2c3b4d5e6f708192a3b4c5d6e7f8",
		"1750000000",
		EmptyBodySHA256Hex,
	)
	got := Sign("test-secret-key", canonical)
	want := "7b8359d13288867df83c6edb43cd3c36823cbde30c6dc2c2ee7b7aed84778850"
	if got != want {
		t.Fatalf("signature mismatch\n got: %s\nwant: %s", got, want)
	}
}

func TestVerifySignature(t *testing.T) {
	sig := Sign("k", "x")
	if !VerifySignature(sig, sig) {
		t.Fatal("equal signatures should verify")
	}
	if VerifySignature(sig, sig[:len(sig)-1]+"0") {
		t.Fatal("tampered signature should not verify")
	}
	if VerifySignature(sig, sig[:len(sig)-1]) {
		t.Fatal("differing length should not verify")
	}
}

func TestBodySHA256Hex(t *testing.T) {
	if BodySHA256Hex(nil) != EmptyBodySHA256Hex {
		t.Fatal("nil body should hash to empty-body constant")
	}
	if BodySHA256Hex([]byte{}) != EmptyBodySHA256Hex {
		t.Fatal("empty body should hash to empty-body constant")
	}
}

func TestGenerateSecret(t *testing.T) {
	s1, err := GenerateSecret()
	if err != nil {
		t.Fatal(err)
	}
	s2, err := GenerateSecret()
	if err != nil {
		t.Fatal(err)
	}
	if s1 == s2 {
		t.Fatal("two GenerateSecret calls returned identical values — CSPRNG broken?")
	}
	if len(s1) != SecretByteLength*2 {
		t.Fatalf("secret hex length: got %d want %d", len(s1), SecretByteLength*2)
	}
	if _, err := hex.DecodeString(s1); err != nil {
		t.Fatalf("secret is not valid hex: %v", err)
	}
}

func TestIsValidNonceFormat(t *testing.T) {
	if !IsValidNonceFormat("9f1a2c3b4d5e6f708192a3b4c5d6e7f8") {
		t.Fatal("valid 32-char hex should pass")
	}
	if IsValidNonceFormat("9f1a2c3b4d5e6f708192a3b4c5d6e7f") {
		t.Fatal("31 chars should fail")
	}
	if IsValidNonceFormat("9f1a2c3b4d5e6f708192a3b4c5d6e7fz") {
		t.Fatal("non-hex char should fail")
	}
}

func TestMaskSecret(t *testing.T) {
	if MaskSecret("") != "" {
		t.Fatal("empty secret masks to empty")
	}
	if MaskSecret("abc") != "****" {
		t.Fatal("short secret masks to fixed length")
	}
	got := MaskSecret("abcdef0123456789")
	if got != "abcd****" {
		t.Fatalf("mask format: got %s", got)
	}
}
