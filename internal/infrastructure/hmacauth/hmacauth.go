package hmacauth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strings"
)

// EmptyBodySHA256Hex 는 빈 바이트열의 SHA-256 hex 값입니다.
// GET 등 본문이 없는 요청의 BODY_SHA256_HEX 필드에 사용됩니다.
const EmptyBodySHA256Hex = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// SecretByteLength 는 발급되는 HMAC 시크릿의 바이트 길이입니다.
// hex 인코딩 시 64자가 됩니다.
const SecretByteLength = 32

// GenerateSecret 은 CSPRNG 로 SecretByteLength 바이트를 생성하고 hex 인코딩하여 반환합니다.
func GenerateSecret() (string, error) {
	buf := make([]byte, SecretByteLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// BodySHA256Hex 는 주어진 바이트열의 SHA-256 hex (소문자) 를 반환합니다.
func BodySHA256Hex(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// BuildCanonical 은 명세에 정의된 canonical 문자열을 구성합니다.
//
//	{METHOD}\n{PATH_AND_QUERY}\n{NONCE}\n{TIMESTAMP}\n{BODY_SHA256_HEX}
//
// METHOD 는 대문자로 정규화됩니다.
func BuildCanonical(method, pathAndQuery, nonce, timestamp, bodySHA256Hex string) string {
	return strings.ToUpper(method) + "\n" +
		pathAndQuery + "\n" +
		nonce + "\n" +
		timestamp + "\n" +
		bodySHA256Hex
}

// Sign 은 secret 으로 canonical 문자열을 HMAC-SHA256 서명한 hex (소문자) 를 반환합니다.
func Sign(secret, canonical string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifySignature 는 기대값과 수신값을 상수시간 비교합니다.
// 길이가 다르면 즉시 false. 같으면 subtle.ConstantTimeCompare.
func VerifySignature(expectedHex, receivedHex string) bool {
	if len(expectedHex) != len(receivedHex) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expectedHex), []byte(receivedHex)) == 1
}

// IsValidNonceFormat 는 nonce 가 32자 소문자 hex 인지 검사합니다.
// 명세상 클라는 UUID v4 의 하이픈을 제거해 32자 hex 로 보냅니다.
func IsValidNonceFormat(nonce string) bool {
	if len(nonce) != 32 {
		return false
	}
	_, err := hex.DecodeString(nonce)
	return err == nil
}

// MaskSecret 은 로깅용으로 시크릿의 앞 4자만 노출합니다.
// 보안상 절대 풀 시크릿을 로그에 남기지 않습니다.
func MaskSecret(secret string) string {
	if secret == "" {
		return ""
	}
	if len(secret) <= 4 {
		return "****"
	}
	return secret[:4] + "****"
}

// ErrInvalidHex 는 hex 디코딩 실패 시 사용됩니다.
var ErrInvalidHex = errors.New("invalid hex string")
