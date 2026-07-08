package authorization

import (
	"strings"
	"time"
)

type Role struct {
	id          uint
	code        string
	name        string
	description string
	createdAt   time.Time
	updatedAt   time.Time
}

type Permission struct {
	id          uint
	code        string
	name        string
	description string
	createdAt   time.Time
	updatedAt   time.Time
}

func NewRole(code, name, description string) (*Role, error) {
	code = normalizeCode(code)
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if code == "" || name == "" {
		return nil, ErrInvalidRole
	}

	return &Role{
		code:        code,
		name:        name,
		description: description,
	}, nil
}

func ReconstituteRole(id uint, code, name, description string, createdAt, updatedAt time.Time) *Role {
	return &Role{
		id:          id,
		code:        code,
		name:        name,
		description: description,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}
}

func (r *Role) Update(name, description string) error {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if name == "" {
		return ErrInvalidRole
	}
	r.name = name
	r.description = description
	return nil
}

func (r *Role) AssignID(id uint) {
	if r.id == 0 {
		r.id = id
	}
}

func (r *Role) GetID() uint             { return r.id }
func (r *Role) GetCode() string         { return r.code }
func (r *Role) GetName() string         { return r.name }
func (r *Role) GetDescription() string  { return r.description }
func (r *Role) GetCreatedAt() time.Time { return r.createdAt }
func (r *Role) GetUpdatedAt() time.Time { return r.updatedAt }

func NewPermission(code, name, description string) (*Permission, error) {
	code = normalizeCode(code)
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if code == "" || name == "" {
		return nil, ErrInvalidPermission
	}

	return &Permission{
		code:        code,
		name:        name,
		description: description,
	}, nil
}

func ReconstitutePermission(id uint, code, name, description string, createdAt, updatedAt time.Time) *Permission {
	return &Permission{
		id:          id,
		code:        code,
		name:        name,
		description: description,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}
}

func (p *Permission) GetID() uint             { return p.id }
func (p *Permission) GetCode() string         { return p.code }
func (p *Permission) GetName() string         { return p.name }
func (p *Permission) GetDescription() string  { return p.description }
func (p *Permission) GetCreatedAt() time.Time { return p.createdAt }
func (p *Permission) GetUpdatedAt() time.Time { return p.updatedAt }

func normalizeCode(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}
