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
