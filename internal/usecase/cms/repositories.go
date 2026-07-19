package cms

import (
	domainCMS "github.com/freeDog-wy/go-backend-template/internal/domain/cms"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
	"time"
)

// Repositories makes each CMS dependency explicit at the composition root.
// The concrete PostgreSQL repository can satisfy every port today, while
// individual use cases can be migrated to narrower dependencies incrementally.
type Repositories struct {
	domainCMS.LocaleRepository
	domainCMS.TagRepository
	domainCMS.CategoryRepository
	domainCMS.ArticleRepository
	domainCMS.ArticleRelationRepository
	domainCMS.RedirectRepository
	domainCMS.PublicContentRepository
}

var _ domainCMS.Repository = Repositories{}

func NewWithRepositories(tx shared.TxManager, repositories Repositories) *Service {
	return &Service{tx: tx, repo: repositories, now: time.Now}
}
