//go:build integration

package cms

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	domainCMS "github.com/freeDog-wy/go-backend-template/internal/domain/cms"
	"github.com/freeDog-wy/go-backend-template/internal/infra/postgres"
	platformAudit "github.com/freeDog-wy/go-backend-template/internal/platform/audit"
	baseRepository "github.com/freeDog-wy/go-backend-template/internal/repository"
	"github.com/freeDog-wy/go-backend-template/internal/testsupport"
	svcCMS "github.com/freeDog-wy/go-backend-template/internal/usecase/cms"
	"gorm.io/gorm"
)

type cmsIntegrationFixture struct {
	ctx      context.Context
	db       *gorm.DB
	repo     *Repository
	authorID uint
}

func newCMSIntegrationFixture(t *testing.T) *cmsIntegrationFixture {
	t.Helper()
	db := testsupport.OpenPostgres(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("database handle: %v", err)
	}
	migrator, err := postgres.NewMigratorWithDB(sqlDB, migrationDir(t))
	if err != nil {
		t.Fatalf("open migrator: %v", err)
	}
	t.Cleanup(func() { _, _ = migrator.Close() })
	if err := migrator.Up(); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	var authorID uint
	if err := db.Raw(`INSERT INTO users (name, email, created_at, updated_at) VALUES ('CMS Author', ?, NOW(), NOW()) RETURNING id`, fmt.Sprintf("cms-it-%d@example.com", time.Now().UnixNano())).Scan(&authorID).Error; err != nil {
		t.Fatalf("create author: %v", err)
	}
	return &cmsIntegrationFixture{ctx: context.Background(), db: db, repo: New(db), authorID: authorID}
}

func (f *cmsIntegrationFixture) service() *svcCMS.Service {
	return svcCMS.NewWithRepositories(baseRepository.NewTxManager(f.db), svcCMS.Repositories{
		LocaleRepository: f.repo, TagRepository: f.repo, CategoryRepository: f.repo, ArticleRepository: f.repo, ArticleRelationRepository: f.repo, RedirectRepository: f.repo, PublicContentRepository: f.repo,
	})
}

func (f *cmsIntegrationFixture) createCategory(t *testing.T, slug string) *domainCMS.Category {
	t.Helper()
	category := &domainCMS.Category{Enabled: true}
	translation := &domainCMS.CategoryTranslation{Locale: "zh-CN", Name: slug, Slug: slug}
	if err := f.repo.CreateCategory(f.ctx, category, translation); err != nil {
		t.Fatalf("create category: %v", err)
	}
	return category
}

func (f *cmsIntegrationFixture) createArticle(t *testing.T, slug string) (*domainCMS.Article, *domainCMS.ArticleTranslation) {
	t.Helper()
	article := &domainCMS.Article{AuthorUserID: f.authorID}
	translation := &domainCMS.ArticleTranslation{Locale: "zh-CN", Title: slug, Slug: slug, ContentFormat: "markdown", Status: domainCMS.TranslationDraft}
	if err := f.repo.CreateArticle(f.ctx, article, translation); err != nil {
		t.Fatalf("create article: %v", err)
	}
	return article, translation
}

type repositoryAuditRecorder struct{ records []platformAudit.RecordInput }

func (r *repositoryAuditRecorder) Record(_ context.Context, input platformAudit.RecordInput) error {
	r.records = append(r.records, input)
	return nil
}

func migrationDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		candidate := filepath.Join(dir, "db", "migrations")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("locate db/migrations from the working directory")
		}
		dir = parent
	}
}
