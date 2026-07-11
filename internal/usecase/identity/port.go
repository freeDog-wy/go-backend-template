package identity

import (
	"context"

	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
)

// RegistrationService registers a user through the public application workflow.
type RegistrationService interface {
	Register(ctx context.Context, cmd RegisterCmd) (*UserResult, error)
}

// ProfileService reads and updates an authenticated user's profile.
type ProfileService interface {
	GetByID(ctx context.Context, userID uint) (*UserResult, error)
	UpdateProfile(ctx context.Context, cmd UpdateProfileCmd) (*UserResult, error)
}

// AdminUserService manages users through administrative application workflows.
type AdminUserService interface {
	List(ctx context.Context, cmd ListUsersCmd) ([]*UserResult, shared.PageResult, error)
	CreateAdminUser(ctx context.Context, cmd CreateAdminUserCmd) (*UserResult, error)
	UpdateStatus(ctx context.Context, cmd UpdateStatusCmd) (*UserResult, error)
}

var (
	_ RegistrationService = (*Service)(nil)
	_ ProfileService      = (*Service)(nil)
	_ AdminUserService    = (*Service)(nil)
)
