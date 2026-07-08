package identity

import (
	"strings"
	"time"

	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
)

type User struct {
	id            uint
	name          string
	email         string
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
	StatusLocked
	StatusBanned
	StatusDeleted
)

// ReconstituteUser 从持久层重建实体——不做业务校验，仅用于 repository 还原。
func ReconstituteUser(
	id uint, name, email string,
	status Status, emailVerified bool,
	lastLoginAt, createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) *User {
	return &User{
		id:            id,
		name:          name,
		email:         email,
		status:        status,
		emailVerified: emailVerified,
		lastLoginAt:   timePtr(lastLoginAt),
		createdAt:     createdAt,
		updatedAt:     updatedAt,
		deletedAt:     deletedAt,
	}
}

func NewUser(name, email string) (*User, error) {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(strings.ToLower(email))
	if name == "" || email == "" {
		return nil, ErrInvalidUserData
	}

	u := &User{
		name:   name,
		email:  email,
		status: StatusPendingVerification,
	}
	u.events = append(u.events, Registered{
		Name:  name,
		Email: email,
	})
	return u, nil
}

func NewAdminUser(name, email string) (*User, error) {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(strings.ToLower(email))
	if name == "" || email == "" {
		return nil, ErrInvalidUserData
	}

	return &User{
		name:          name,
		email:         email,
		status:        StatusActive,
		emailVerified: true,
	}, nil
}

func (u *User) Activate() error {
	switch u.status {
	case StatusPendingVerification:
		u.status = StatusActive
	case StatusLocked:
		return ErrUserLocked
	case StatusBanned:
		return ErrUserBanned
	case StatusDeleted:
		return ErrUserDeleted
	}
	return nil
}

func (u *User) Lock() error {
	switch u.status {
	case StatusBanned:
		return ErrUserBanned
	case StatusDeleted:
		return ErrUserDeleted
	}
	u.status = StatusLocked
	return nil
}

func (u *User) Ban() {
	u.status = StatusBanned
}

func (u *User) Delete(at time.Time) {
	u.status = StatusDeleted
	u.deletedAt = &at
}

func (u *User) UpdateProfile(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidUserData
	}
	u.name = name
	return nil
}

func (u *User) IsActive() bool              { return u.status == StatusActive }
func (u *User) IsLocked() bool              { return u.status == StatusLocked }
func (u *User) IsBanned() bool              { return u.status == StatusBanned }
func (u *User) IsDeleted() bool             { return u.status == StatusDeleted }
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

func (u *User) MarkPersisted(createdAt, updatedAt time.Time) {
	if !createdAt.IsZero() {
		u.createdAt = createdAt
	}
	if !updatedAt.IsZero() {
		u.updatedAt = updatedAt
	}
}

func (u *User) GetID() uint       { return u.id }
func (u *User) GetName() string   { return u.name }
func (u *User) GetEmail() string  { return u.email }
func (u *User) GetStatus() Status { return u.status }
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
