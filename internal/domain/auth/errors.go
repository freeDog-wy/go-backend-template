package auth

import "errors"

var (
	ErrInvalidCredential = errors.New("invalid credential")
	ErrInvalidSession    = errors.New("invalid session")
	ErrInvalidClaims     = errors.New("invalid claims")
)
