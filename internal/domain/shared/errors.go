package shared

import "fmt"

var (
	ErrNotFound     = fmt.Errorf("not found")
	ErrUnauthorized = fmt.Errorf("unauthorized")
	ErrInvalidInput = fmt.Errorf("invalid input")
)
