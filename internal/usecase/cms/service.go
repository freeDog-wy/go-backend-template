package cms

import (
	"context"
	"time"

	domainCMS "github.com/freeDog-wy/go-backend-template/internal/domain/cms"
	domainMedia "github.com/freeDog-wy/go-backend-template/internal/domain/media"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
	platformAudit "github.com/freeDog-wy/go-backend-template/internal/platform/audit"
)

// Service coordinates CMS writes, public queries, audit records, and media checks.
//
// Public slugs, redirects, and audit records must remain in the same transaction as
// their content updates. Read operations must not have external side effects.
type Service struct {
	tx                shared.TxManager
	repo              Repositories
	now               func() time.Time
	auditor           platformAudit.Recorder
	mediaFinder       ReadyMediaFinder
	publicMediaFinder PublicMediaFinder
}

type ReadyMediaFinder interface {
	IsReady(context.Context, uint) (bool, error)
}

type PublicMediaFinder interface {
	ListPublic(context.Context, string, []uint) ([]domainMedia.PublicAsset, error)
}

func New(tx shared.TxManager, repo domainCMS.Repository) *Service {
	return NewWithRepositories(tx, Repositories{
		LocaleRepository:          repo,
		TagRepository:             repo,
		CategoryRepository:        repo,
		ArticleRepository:         repo,
		ArticleRelationRepository: repo,
		RedirectRepository:        repo,
		PublicContentRepository:   repo,
	})
}

func (s *Service) SetMediaFinder(f ReadyMediaFinder)        { s.mediaFinder = f }
func (s *Service) SetPublicMediaFinder(f PublicMediaFinder) { s.publicMediaFinder = f }

func (s *Service) SetAuditRecorder(recorder platformAudit.Recorder) {
	if recorder != nil {
		s.auditor = recorder
	}
}
