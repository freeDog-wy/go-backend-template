package verification

import "fmt"

var (
	ErrInvalidEmailVerificationToken = fmt.Errorf("invalid email verification token")
	ErrInvalidPasswordResetToken     = fmt.Errorf("invalid password reset token")
)
