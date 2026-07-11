package verification

import "context"

// VerificationService manages email-verification and password-reset workflows.
type VerificationService interface {
	ResendVerification(ctx context.Context, cmd ResendVerificationCmd) error
	VerifyEmail(ctx context.Context, cmd VerifyEmailCmd) error
	ForgotPassword(ctx context.Context, cmd ForgotPasswordCmd) error
	ResetPassword(ctx context.Context, cmd ResetPasswordCmd) error
}

var _ VerificationService = (*Service)(nil)
