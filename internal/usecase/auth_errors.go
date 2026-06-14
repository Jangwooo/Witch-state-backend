package usecase

import "errors"

// authError represents authentication/authorization failures that should map to 401.
type authError struct {
	msg string
}

func (e *authError) Error() string {
	return e.msg
}

func newAuthError(msg string) error {
	return &authError{msg: msg}
}

// IsAuthError reports whether err is an authentication/authorization error.
func IsAuthError(err error) bool {
	var authErr *authError
	return errors.As(err, &authErr)
}

// bannedError 는 users.is_banned=true 인 계정 거절을 표현합니다. 403 으로 매핑.
type bannedError struct{}

func (e *bannedError) Error() string {
	return "account_banned"
}

// ErrAccountBanned 는 비교 가능한 sentinel 값입니다.
var ErrAccountBanned error = &bannedError{}

// IsBannedError 는 err 가 차단 계정 거절 에러인지 반환합니다.
func IsBannedError(err error) bool {
	var be *bannedError
	return errors.As(err, &be)
}
