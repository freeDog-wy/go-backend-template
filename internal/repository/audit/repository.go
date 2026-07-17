package audit

import (
	"context"

	domainAudit "github.com/freeDog-wy/go-backend-template/internal/domain/audit"
	modelAudit "github.com/freeDog-wy/go-backend-template/internal/model/audit"
	repositorytx "github.com/freeDog-wy/go-backend-template/internal/repository"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

var _ domainAudit.Repository = (*Repository)(nil)

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) g(ctx context.Context) gorm.Interface[modelAudit.Log] {
	return gorm.G[modelAudit.Log](repositorytx.DB(ctx, r.db))
}

func (r *Repository) Create(ctx context.Context, log *domainAudit.AuditLog) error {
	return r.g(ctx).Create(ctx, modelAudit.FromEntity(log))
}
