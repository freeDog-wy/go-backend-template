package authorization

import "time"

type RolePermission struct {
	RoleID       uint      `gorm:"primaryKey"`
	PermissionID uint      `gorm:"primaryKey"`
	CreatedAt    time.Time `gorm:"not null"`
}
