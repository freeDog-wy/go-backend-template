package cms

import (
	"context"
	"errors"

	domainCMS "github.com/freeDog-wy/go-backend-template/internal/domain/cms"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
	repositorytx "github.com/freeDog-wy/go-backend-template/internal/repository"
	"gorm.io/gorm"
)

// Repository is the PostgreSQL implementation shared by the CMS store ports.
// Transaction ownership remains with the use case through the context-bound DB.
type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) conn(ctx context.Context) *gorm.DB {
	return repositorytx.DB(ctx, r.db)
}

func mapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return shared.ErrNotFound
	}
	return err
}

var (
	_ domainCMS.LocaleRepository          = (*Repository)(nil)
	_ domainCMS.TagRepository             = (*Repository)(nil)
	_ domainCMS.CategoryRepository        = (*Repository)(nil)
	_ domainCMS.ArticleRepository         = (*Repository)(nil)
	_ domainCMS.ArticleRelationRepository = (*Repository)(nil)
	_ domainCMS.RedirectRepository        = (*Repository)(nil)
	_ domainCMS.PublicContentRepository   = (*Repository)(nil)
)
