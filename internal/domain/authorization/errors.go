package authorization

import "errors"

var (
	ErrInvalidRole       = errors.New("invalid role")
	ErrInvalidPermission = errors.New("invalid permission")
	ErrRoleNotFound      = errors.New("role not found")
	ErrPermissionDenied  = errors.New("permission denied")
)
