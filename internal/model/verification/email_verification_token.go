package verification

import (
	"time"

	domainVerification "github.com/freeDog-wy/go-backend-template/internal/domain/verification"
)

type EmailVerificationToken struct {
	ID         uint       `gorm:"primaryKey"`
	UserID     uint       `gorm:"index;not null"`
	TokenHash  string     `gorm:"type:char(64);uniqueIndex;not null"`
	ExpiresAt  time.Time  `gorm:"index;not null"`
	ConsumedAt *time.Time `gorm:"index"`
	CreatedAt  time.Time  `gorm:"not null"`
}

func (t *EmailVerificationToken) ToEntity() *domainVerification.EmailVerificationToken {
	return domainVerification.ReconstituteEmailVerificationToken(
		t.ID,
		t.UserID,
		t.TokenHash,
		t.ExpiresAt,
		t.ConsumedAt,
		t.CreatedAt,
	)
}

func FromEntity(e *domainVerification.EmailVerificationToken) *EmailVerificationToken {
	return &EmailVerificationToken{
		ID:         e.GetID(),
		UserID:     e.GetUserID(),
		TokenHash:  e.GetTokenHash(),
		ExpiresAt:  e.GetExpiresAt(),
		ConsumedAt: e.GetConsumedAt(),
		CreatedAt:  e.GetCreatedAt(),
	}
}
