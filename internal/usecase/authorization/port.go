package authorization

import (
	"context"

	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
)

// AccessAuthorizer evaluates administrative access and named permissions.
type AccessAuthorizer interface {
	EnsureAdminAccess(ctx context.Context, userID uint) error
	HasPermission(ctx context.Context, userID uint, code string) (bool, error)
}

// RoleService manages role and permission definitions.
type RoleService interface {
	ListRoles(ctx context.Context, cmd ListRolesCmd) ([]*RoleResult, shared.PageResult, error)
	CreateRole(ctx context.Context, cmd CreateRoleCmd) (*RoleResult, error)
	UpdateRole(ctx context.Context, cmd UpdateRoleCmd) (*RoleResult, error)
	ListPermissions(ctx context.Context, cmd ListPermissionsCmd) ([]*PermissionResult, shared.PageResult, error)
}

// UserRoleService manages a user's role bindings.
type UserRoleService interface {
	ReplaceUserRoles(ctx context.Context, cmd ReplaceUserRolesCmd) error
	ListUserRoles(ctx context.Context, userID uint) ([]*RoleResult, error)
}

var (
	_ AccessAuthorizer = (*Service)(nil)
	_ RoleService      = (*Service)(nil)
	_ UserRoleService  = (*Service)(nil)
)
