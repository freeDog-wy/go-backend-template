package authorization

import (
	"time"

	domainAuthorization "github.com/freeDog-wy/go-backend-template/internal/domain/authorization"
)

type Role struct {
	ID          uint      `gorm:"primaryKey"`
	Code        string    `gorm:"type:varchar(100);uniqueIndex;not null"`
	Name        string    `gorm:"type:varchar(100);not null"`
	Description string    `gorm:"type:text"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (r *Role) ToEntity() *domainAuthorization.Role {
	return domainAuthorization.ReconstituteRole(
		r.ID,
		r.Code,
		r.Name,
		r.Description,
		r.CreatedAt,
		r.UpdatedAt,
	)
}

func RoleFromEntity(e *domainAuthorization.Role) *Role {
	return &Role{
		ID:          e.GetID(),
		Code:        e.GetCode(),
		Name:        e.GetName(),
		Description: e.GetDescription(),
		CreatedAt:   e.GetCreatedAt(),
		UpdatedAt:   e.GetUpdatedAt(),
	}
}
