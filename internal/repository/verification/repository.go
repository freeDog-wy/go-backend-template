package verification

import (
	"context"
	"errors"
	"time"

	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
	domainVerification "github.com/freeDog-wy/go-backend-template/internal/domain/verification"
	"github.com/freeDog-wy/go-backend-template/internal/infra/database"
	modelVerification "github.com/freeDog-wy/go-backend-template/internal/model/verification"

	"gorm.io/gorm"
)

// Repository 实现 domain/verification.Repository。
type Repository struct {
	db *gorm.DB
}

var _ domainVerification.Repository = (*Repository)(nil)

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) g(ctx context.Context) gorm.Interface[modelVerification.EmailVerificationToken] {
	return gorm.G[modelVerification.EmailVerificationToken](database.DB(ctx, r.db))
}

func (r *Repository) Create(ctx context.Context, token *domainVerification.EmailVerificationToken) error {
	m := modelVerification.FromEntity(token)
	return r.g(ctx).Create(ctx, m)
}

func (r *Repository) FindActiveByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*domainVerification.EmailVerificationToken, error) {
	m, err := r.g(ctx).
		Where("token_hash = ? AND consumed_at IS NULL AND expires_at > ?", tokenHash, now).
		First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return m.ToEntity(), nil
}

func (r *Repository) InvalidateByUserID(ctx context.Context, userID uint, now time.Time) error {
	return database.DB(ctx, r.db).
		Model(&modelVerification.EmailVerificationToken{}).
		Where("user_id = ? AND consumed_at IS NULL", userID).
		Update("consumed_at", now).Error
}

func (r *Repository) Update(ctx context.Context, token *domainVerification.EmailVerificationToken) error {
	m := modelVerification.FromEntity(token)
	return database.DB(ctx, r.db).
		Model(&modelVerification.EmailVerificationToken{}).
		Where("id = ?", m.ID).
		Updates(map[string]any{
			"consumed_at": m.ConsumedAt,
		}).Error
}
