package identity

import (
	"time"

	domainIdentity "github.com/freeDog-wy/go-backend-template/internal/domain/identity"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name          string     `gorm:"type:varchar(100);not null"`
	Email         string     `gorm:"type:varchar(100);unique;not null"`
	PasswordHash  string     `gorm:"type:varchar(255);not null"`
	EmailVerified bool       `gorm:"type:boolean;default:false"`
	LastLoginAt   *time.Time `gorm:"column:last_login_at"`
	Status        int        `gorm:"type:smallint;default:0;not null"`
}

func (u *User) ToEntity() *domainIdentity.User {
	var deletedAt *time.Time
	if u.DeletedAt.Valid {
		deletedAt = &u.DeletedAt.Time
	}
	return domainIdentity.ReconstituteUser(
		u.ID,
		u.Name,
		u.Email,
		u.PasswordHash,
		domainIdentity.Status(u.Status),
		u.EmailVerified,
		timeOrZero(u.LastLoginAt),
		u.CreatedAt,
		u.UpdatedAt,
		deletedAt,
	)
}

// FromEntity 将领域实体转换为数据库模型。
func FromEntity(e *domainIdentity.User) *User {
	return &User{
		Model: gorm.Model{
			ID:        e.GetID(),
			CreatedAt: time.Time{},
			UpdatedAt: time.Time{},
		},
		Name:          e.GetName(),
		Email:         e.GetEmail(),
		PasswordHash:  e.GetPasswordHash(),
		EmailVerified: e.IsEmailVerified(),
		LastLoginAt:   e.GetLastLoginAt(),
		Status:        int(e.GetStatus()),
	}
}

func timeOrZero(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}
