package identity

import (
	"context"

	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
)

type Repository interface {
	FindByID(ctx context.Context, id uint) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	List(ctx context.Context, page shared.PageQuery) ([]*User, int64, error)
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uint) error
}
