package user

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
	events        []shared.Event // 领域事件，由 factory 方法记录，service 层读取后发布
}

type Status int

const (
	StatusInactive Status = iota
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
		status:       StatusInactive,
	}
	u.events = append(u.events, Registered{
		UserID: 0,
		Name:   name,
		Email:  email,
	})
	return u, nil
}

// —————————— 行为方法 ——————————

func (u *User) Activate() error {
	switch u.status {
	case StatusInactive:
		u.status = StatusActive
	case StatusBanned:
		return ErrUserBanned
	}
	return nil
}

func (u *User) Ban() {
	u.status = StatusBanned
}

func (u *User) IsActive() bool  { return u.status == StatusActive }
func (u *User) IsBanned() bool  { return u.status == StatusBanned }

// VerifyEmail 标记邮箱已验证。一次验证后不可逆。
func (u *User) VerifyEmail() {
	u.emailVerified = true
}

func (u *User) IsEmailVerified() bool { return u.emailVerified }

// RecordLogin 记录最近一次登录时间。
func (u *User) RecordLogin(t time.Time) {
	u.lastLoginAt = &t
}

// —————————— Getter ——————————

func (u *User) GetID() uint              { return u.id }
func (u *User) GetName() string          { return u.name }
func (u *User) GetEmail() string         { return u.email }
func (u *User) GetPasswordHash() string    { return u.passwordHash }
func (u *User) GetStatus() Status           { return u.status }
func (u *User) GetLastLoginAt() *time.Time  { return u.lastLoginAt }

// Events 返回实体生命周期内记录的领域事件。
func (u *User) Events() []shared.Event { return u.events }

// ClearEvents 清空事件列表——service 发布后调用，防止重复发布。
func (u *User) ClearEvents() { u.events = nil }

// —————————— 内部工具 ——————————

func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
