package user

import "fmt"

var (
	ErrInvalidUserData   = fmt.Errorf("invalid user data")
	ErrUserNotFound      = fmt.Errorf("user not found")
	ErrUserAlreadyExists = fmt.Errorf("user already exists")
	ErrUserBanned        = fmt.Errorf("user is banned")
)
