package verification

import (
	"context"
	"time"
)

type Repository interface {
	Create(ctx context.Context, token *EmailVerificationToken) error
	FindActiveByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*EmailVerificationToken, error)
	InvalidateByUserID(ctx context.Context, userID uint, now time.Time) error
	Update(ctx context.Context, token *EmailVerificationToken) error
}
