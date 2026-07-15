package serviceaccount

import "context"

type Repository interface {
	Create(ctx context.Context, account *ServiceAccount) error
	Update(ctx context.Context, account *ServiceAccount) error
	FindByClientID(ctx context.Context, clientID string) (*ServiceAccount, error)
	FindByUserID(ctx context.Context, userID uint) (*ServiceAccount, error)
}
