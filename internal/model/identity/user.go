package identity

import (
	"time"

	domainIdentity "github.com/freeDog-wy/go-backend-template/internal/domain/identity"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name                string     `gorm:"type:varchar(100);not null"`
	Email               string     `gorm:"type:varchar(100);unique;not null"`
	EmailVerified       bool       `gorm:"type:boolean;default:false"`
	FailedLoginAttempts int        `gorm:"column:failed_login_attempts;default:0;not null"`
	LastLoginAt         *time.Time `gorm:"column:last_login_at"`
	Status              int        `gorm:"type:smallint;default:0;not null"`
}

func (u *User) ToEntity() *domainIdentity.User {
	var deletedAt *time.Time
	if u.DeletedAt.Valid {
		deletedAt = &u.DeletedAt.Time
	}
	return domainIdentity.ReconstituteUserWithLoginFailures(
		u.ID,
		u.Name,
		u.Email,
		domainIdentity.Status(u.Status),
		u.EmailVerified,
		u.FailedLoginAttempts,
		timeOrZero(u.LastLoginAt),
		u.CreatedAt,
		u.UpdatedAt,
		deletedAt,
	)
}

func FromEntity(e *domainIdentity.User) *User {
	return &User{
		Model: gorm.Model{
			ID:        e.GetID(),
			CreatedAt: time.Time{},
			UpdatedAt: time.Time{},
		},
		Name:                e.GetName(),
		Email:               e.GetEmail(),
		EmailVerified:       e.IsEmailVerified(),
		FailedLoginAttempts: e.GetFailedLoginAttempts(),
		LastLoginAt:         e.GetLastLoginAt(),
		Status:              int(e.GetStatus()),
	}
}

func timeOrZero(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}
