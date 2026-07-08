package auth

import (
	"time"

	domainAuth "github.com/freeDog-wy/go-backend-template/internal/domain/auth"
)

type UserCredential struct {
	UserID            uint      `gorm:"primaryKey"`
	PasswordHash      string    `gorm:"type:varchar(255);not null"`
	PasswordChangedAt time.Time `gorm:"not null"`
	CreatedAt         time.Time `gorm:"not null"`
	UpdatedAt         time.Time `gorm:"not null"`
}

func (c *UserCredential) ToEntity() *domainAuth.UserCredential {
	return domainAuth.ReconstituteUserCredential(
		c.UserID,
		c.PasswordHash,
		c.PasswordChangedAt,
		c.CreatedAt,
		c.UpdatedAt,
	)
}

func FromEntity(e *domainAuth.UserCredential) *UserCredential {
	return &UserCredential{
		UserID:            e.GetUserID(),
		PasswordHash:      e.GetPasswordHash(),
		PasswordChangedAt: e.GetPasswordChangedAt(),
		CreatedAt:         e.GetCreatedAt(),
		UpdatedAt:         e.GetUpdatedAt(),
	}
}
