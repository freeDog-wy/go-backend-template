package identity

import (
	"time"

	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
)

type User struct {
	id            uint
	name          string
	email         string
	passwordHash  string
	status        Status
	emailVerified bool
	lastLoginAt   *time.Time
	createdAt     time.Time
	updatedAt     time.Time
	deletedAt     *time.Time
	events        []shared.Event
}

type Status int

const (
	StatusPendingVerification Status = iota
	StatusActive
	StatusBanned
)

// ReconstituteUser 从持久层重建实体——不做业务校验，仅用于 repository 还原。
func ReconstituteUser(
	id uint, name, email, passwordHash string,
	status Status, emailVerified bool,
	lastLoginAt, createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) *User {
	return &User{
		id:            id,
		name:          name,
		email:         email,
		passwordHash:  passwordHash,
		status:        status,
		emailVerified: emailVerified,
		lastLoginAt:   timePtr(lastLoginAt),
		createdAt:     createdAt,
		updatedAt:     updatedAt,
		deletedAt:     deletedAt,
	}
}

func NewUser(name, email, passwordHash string) (*User, error) {
	if name == "" || email == "" || passwordHash == "" {
		return nil, ErrInvalidUserData
	}

	u := &User{
		name:         name,
		email:        email,
		passwordHash: passwordHash,
		status:       StatusPendingVerification,
	}
	u.events = append(u.events, Registered{
		Name:  name,
		Email: email,
	})
	return u, nil
}

func (u *User) Activate() error {
	switch u.status {
	case StatusPendingVerification:
		u.status = StatusActive
	case StatusBanned:
		return ErrUserBanned
	}
	return nil
}

func (u *User) Ban() {
	u.status = StatusBanned
}

func (u *User) IsActive() bool              { return u.status == StatusActive }
func (u *User) IsBanned() bool              { return u.status == StatusBanned }
func (u *User) IsPendingVerification() bool { return u.status == StatusPendingVerification }

// VerifyEmail 标记邮箱已验证。一次验证后不可逆。
func (u *User) VerifyEmail() {
	u.emailVerified = true
}

func (u *User) IsEmailVerified() bool { return u.emailVerified }

// RecordLogin 记录最近一次登录时间。
func (u *User) RecordLogin(t time.Time) {
	u.lastLoginAt = &t
}

// AssignID 在实体首次持久化后回填主键，并同步修正待发布事件中的聚合 ID。
func (u *User) AssignID(id uint) {
	if id == 0 || u.id == id {
		return
	}

	u.id = id
	for i, evt := range u.events {
		registered, ok := evt.(Registered)
		if !ok {
			continue
		}

		registered.UserID = id
		u.events[i] = registered
	}
}

func (u *User) GetID() uint             { return u.id }
func (u *User) GetName() string         { return u.name }
func (u *User) GetEmail() string        { return u.email }
func (u *User) GetPasswordHash() string { return u.passwordHash }
func (u *User) GetStatus() Status       { return u.status }
func (u *User) GetLastLoginAt() *time.Time {
	return u.lastLoginAt
}

func (u *User) Events() []shared.Event { return u.events }
func (u *User) ClearEvents()           { u.events = nil }

func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
