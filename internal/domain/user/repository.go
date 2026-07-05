package user

import "context"

type Repository interface {
	FindByID(ctx context.Context, id uint) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uint) error
}
