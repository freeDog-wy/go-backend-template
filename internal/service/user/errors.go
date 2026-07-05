package user

import "errors"

var (
	ErrInvalidCaptcha = errors.New("invalid captcha")
	ErrEmailTaken     = errors.New("email already taken")
)
