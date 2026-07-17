package identity

import (
	"context"
	"errors"

	domainIdentity "github.com/freeDog-wy/go-backend-template/internal/domain/identity"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
	modelIdentity "github.com/freeDog-wy/go-backend-template/internal/model/identity"
	repositorytx "github.com/freeDog-wy/go-backend-template/internal/repository"

	"gorm.io/gorm"
)

// Repository 实现 domain/identity.Repository，基于 GORM 泛型 API。
type Repository struct {
	db *gorm.DB
}

var _ domainIdentity.Repository = (*Repository)(nil)

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// g 返回类型安全的泛型链，自动适配事务/非事务。
func (r *Repository) g(ctx context.Context) gorm.Interface[modelIdentity.User] {
	return gorm.G[modelIdentity.User](repositorytx.DB(ctx, r.db))
}

func (r *Repository) FindByID(ctx context.Context, id uint) (*domainIdentity.User, error) {
	m, err := r.g(ctx).Where("id = ?", id).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return m.ToEntity(), nil
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (*domainIdentity.User, error) {
	m, err := r.g(ctx).Where("email = ?", email).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return m.ToEntity(), nil
}

func (r *Repository) List(ctx context.Context, page shared.PageQuery) ([]*domainIdentity.User, int64, error) {
	var total int64
	if err := repositorytx.DB(ctx, r.db).Model(&modelIdentity.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	models, err := r.g(ctx).
		Order("id ASC").
		Limit(page.PerPage).
		Offset(page.Offset()).
		Find(ctx)
	if err != nil {
		return nil, 0, err
	}
	users := make([]*domainIdentity.User, 0, len(models))
	for i := range models {
		users = append(users, models[i].ToEntity())
	}
	return users, total, nil
}

func (r *Repository) Create(ctx context.Context, user *domainIdentity.User) error {
	m := modelIdentity.FromEntity(user)
	if err := r.g(ctx).Create(ctx, m); err != nil {
		return err
	}

	user.AssignID(m.ID)
	user.MarkPersisted(m.CreatedAt, m.UpdatedAt)
	return nil
}

func (r *Repository) Update(ctx context.Context, user *domainIdentity.User) error {
	m := modelIdentity.FromEntity(user)
	_, err := r.g(ctx).Where("id = ?", m.ID).Updates(ctx, *m)
	return err
}

func (r *Repository) Delete(ctx context.Context, id uint) error {
	_, err := r.g(ctx).Where("id = ?", id).Delete(ctx)
	return err
}
