package auth

import "errors"

var (
	ErrInvalidCaptcha           = errors.New("invalid captcha")
	ErrEmailTaken               = errors.New("email already taken")
	ErrInvalidVerificationToken = errors.New("invalid verification token")
)
