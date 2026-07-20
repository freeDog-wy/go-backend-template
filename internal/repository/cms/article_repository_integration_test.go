//go:build integration

package cms

import (
	"errors"
	"testing"
	"time"

	domainCMS "github.com/freeDog-wy/go-backend-template/internal/domain/cms"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
	modelCMS "github.com/freeDog-wy/go-backend-template/internal/model/cms"
	svcCMS "github.com/freeDog-wy/go-backend-template/internal/usecase/cms"
)

func TestArticleRepositoryIntegrationConstraintsAndLifecycle(t *testing.T) {
	fixture := newCMSIntegrationFixture(t)
	category := fixture.createCategory(t, "root")
	draftArticle, draft := fixture.createArticle(t, "draft")
	publishedArticle, published := fixture.createArticle(t, "published")
	now := time.Now().UTC().Add(-time.Second)
	published.Status, published.PublishedAt = domainCMS.TranslationPublished, &now
	if err := fixture.repo.SaveArticleTranslation(fixture.ctx, published); err != nil {
		t.Fatalf("publish translation: %v", err)
	}
	if err := fixture.repo.ReplaceArticleCategories(fixture.ctx, draftArticle.ID, []uint{category.ID}, &category.ID); err != nil {
		t.Fatalf("set primary category: %v", err)
	}
	if err := fixture.db.Create(&modelCMS.ArticleCategory{ArticleID: draftArticle.ID, CategoryID: category.ID + 1000, IsPrimary: true}).Error; err == nil {
		t.Fatal("expected primary category constraint error")
	}
	duplicate := &domainCMS.ArticleTranslation{ArticleID: publishedArticle.ID, Locale: "zh-CN", Title: "Duplicate", Slug: draft.Slug, ContentFormat: "markdown", Status: domainCMS.TranslationDraft}
	if err := fixture.repo.CreateArticleTranslation(fixture.ctx, duplicate); err == nil {
		t.Fatal("expected locale slug uniqueness error")
	}
	publishedItems, publishedTotal, err := fixture.repo.ListArticleTranslations(fixture.ctx, "zh-CN", domainCMS.TranslationPublished, false, shared.NewPageQuery(1, 20))
	if err != nil || publishedTotal != 1 || len(publishedItems) != 1 || publishedItems[0].Article.ID != publishedArticle.ID {
		t.Fatalf("published articles = %#v, total = %d, err = %v", publishedItems, publishedTotal, err)
	}

	recorder := &repositoryAuditRecorder{}
	service := fixture.service()
	service.SetAuditRecorder(recorder)
	if err := service.DeleteArticle(fixture.ctx, svcCMS.DeleteArticleCmd{ArticleID: publishedArticle.ID, ActorUserID: fixture.authorID}); err != nil {
		t.Fatalf("delete article: %v", err)
	}
	if _, err := fixture.repo.FindPublicArticle(fixture.ctx, "zh-CN", published.Slug); !errors.Is(err, shared.ErrNotFound) {
		t.Fatalf("deleted public lookup error = %v", err)
	}
	if len(recorder.records) != 1 || recorder.records[0].Action != "cms_article_deleted" {
		t.Fatalf("audit records = %#v, want one article delete record", recorder.records)
	}
	if err := service.RestoreArticle(fixture.ctx, svcCMS.RestoreArticleCmd{ArticleID: publishedArticle.ID, ActorUserID: fixture.authorID}); err != nil {
		t.Fatalf("restore article: %v", err)
	}
	if got, err := fixture.repo.FindPublicArticle(fixture.ctx, "zh-CN", published.Slug); err != nil || got.Article.ID != publishedArticle.ID {
		t.Fatalf("restored public article = %#v, %v", got, err)
	}
}
