//go:build integration

package cms

import (
	"testing"
	"time"

	domainCMS "github.com/freeDog-wy/go-backend-template/internal/domain/cms"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
	svcCMS "github.com/freeDog-wy/go-backend-template/internal/usecase/cms"
)

func TestRedirectRepositoryIntegrationSlugChanges(t *testing.T) {
	fixture := newCMSIntegrationFixture(t)
	category := fixture.createCategory(t, "root")
	article, translation := fixture.createArticle(t, "published")
	now := time.Now().UTC().Add(-time.Second)
	translation.Status, translation.PublishedAt = domainCMS.TranslationPublished, &now
	if err := fixture.repo.SaveArticleTranslation(fixture.ctx, translation); err != nil {
		t.Fatalf("publish translation: %v", err)
	}

	service := fixture.service()
	oldArticleSlug := translation.Slug
	if _, err := service.UpdateTranslation(fixture.ctx, svcCMS.UpdateTranslationCmd{ArticleID: article.ID, Locale: "zh-CN", Title: translation.Title, Slug: "published-renamed", ContentFormat: translation.ContentFormat}); err != nil {
		t.Fatalf("rename article slug: %v", err)
	}
	if _, err := service.UpdateTranslation(fixture.ctx, svcCMS.UpdateTranslationCmd{ArticleID: article.ID, Locale: "zh-CN", Title: translation.Title, Slug: "published-final", ContentFormat: translation.ContentFormat}); err != nil {
		t.Fatalf("rename article slug again: %v", err)
	}
	if redirect, err := fixture.repo.FindURLRedirect(fixture.ctx, "zh-CN", "/zh-CN/articles/"+oldArticleSlug); err != nil || redirect.TargetPath != "/zh-CN/articles/published-final" {
		t.Fatalf("article redirect = %#v, %v", redirect, err)
	}
	if _, err := service.UpsertCategoryTranslation(fixture.ctx, svcCMS.UpsertCategoryTranslationCmd{CategoryID: category.ID, Locale: "zh-CN", Name: "Root", Slug: "root-renamed"}); err != nil {
		t.Fatalf("rename category slug: %v", err)
	}
	if redirect, err := fixture.repo.FindURLRedirect(fixture.ctx, "zh-CN", "/zh-CN/categories/root"); err != nil || redirect.TargetPath != "/zh-CN/categories/root-renamed" {
		t.Fatalf("category redirect = %#v, %v", redirect, err)
	}
	if redirects, total, err := fixture.repo.ListURLRedirects(fixture.ctx, "zh-CN", shared.NewPageQuery(1, 20)); err != nil || total != 3 || len(redirects) != 3 {
		t.Fatalf("redirects=%#v total=%d err=%v", redirects, total, err)
	}
}
