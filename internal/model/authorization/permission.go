package authorization

import (
	"time"

	domainAuthorization "github.com/freeDog-wy/go-backend-template/internal/domain/authorization"
)

type Permission struct {
	ID          uint      `gorm:"primaryKey"`
	Code        string    `gorm:"type:varchar(100);uniqueIndex;not null"`
	Name        string    `gorm:"type:varchar(100);not null"`
	Description string    `gorm:"type:text"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (p *Permission) ToEntity() *domainAuthorization.Permission {
	return domainAuthorization.ReconstitutePermission(
		p.ID,
		p.Code,
		p.Name,
		p.Description,
		p.CreatedAt,
		p.UpdatedAt,
	)
}

func PermissionFromEntity(e *domainAuthorization.Permission) *Permission {
	return &Permission{
		ID:          e.GetID(),
		Code:        e.GetCode(),
		Name:        e.GetName(),
		Description: e.GetDescription(),
		CreatedAt:   e.GetCreatedAt(),
		UpdatedAt:   e.GetUpdatedAt(),
	}
}
