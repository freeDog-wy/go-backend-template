package identity

import "fmt"

var (
	ErrInvalidUserData   = fmt.Errorf("invalid user data")
	ErrUserNotFound      = fmt.Errorf("user not found")
	ErrUserAlreadyExists = fmt.Errorf("user already exists")
	ErrUserLocked        = fmt.Errorf("user is locked")
	ErrUserBanned        = fmt.Errorf("user is banned")
	ErrUserDeleted       = fmt.Errorf("user is deleted")
)
