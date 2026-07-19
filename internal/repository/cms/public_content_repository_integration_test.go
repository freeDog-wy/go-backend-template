//go:build integration

package cms

import (
	"errors"
	"testing"
	"time"

	domainCMS "github.com/freeDog-wy/go-backend-template/internal/domain/cms"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
	modelCMS "github.com/freeDog-wy/go-backend-template/internal/model/cms"
)

func TestPublicContentRepositoryIntegrationVisibilityAndTaxonomy(t *testing.T) {
	fixture := newCMSIntegrationFixture(t)
	category := fixture.createCategory(t, "root")
	_, draft := fixture.createArticle(t, "draft-only")
	if _, err := fixture.repo.FindPublicArticle(fixture.ctx, "zh-CN", draft.Slug); !errors.Is(err, shared.ErrNotFound) {
		t.Fatalf("draft public lookup error = %v", err)
	}

	article, translation := fixture.createArticle(t, "published")
	now := time.Now().UTC().Add(-time.Second)
	translation.Status, translation.PublishedAt = domainCMS.TranslationPublished, &now
	if err := fixture.repo.SaveArticleTranslation(fixture.ctx, translation); err != nil {
		t.Fatalf("publish translation: %v", err)
	}
	if err := fixture.repo.ReplaceArticleCategories(fixture.ctx, article.ID, []uint{category.ID}, &category.ID); err != nil {
		t.Fatalf("set published article category: %v", err)
	}
	tag := &domainCMS.Tag{}
	if err := fixture.repo.CreateTag(fixture.ctx, tag, &domainCMS.TagTranslation{Locale: "zh-CN", Name: "Go", Slug: "go"}); err != nil {
		t.Fatalf("create tag: %v", err)
	}
	if err := fixture.repo.ReplaceArticleTags(fixture.ctx, article.ID, []uint{tag.ID}); err != nil {
		t.Fatalf("attach tag: %v", err)
	}
	if tagged, total, err := fixture.repo.ListPublicTagArticles(fixture.ctx, "zh-CN", "go", shared.NewPageQuery(1, 20)); err != nil || total != 1 || len(tagged) != 1 || tagged[0].Article.ID != article.ID {
		t.Fatalf("tag articles=%#v total=%d err=%v", tagged, total, err)
	}
	if tags, total, err := fixture.repo.ListPublicTags(fixture.ctx, "zh-CN", shared.NewPageQuery(1, 20)); err != nil || total != 1 || len(tags) != 1 || tags[0].Slug != "go" {
		t.Fatalf("public tags=%#v total=%d err=%v", tags, total, err)
	}
	if entries, total, err := fixture.repo.ListPublicSitemapEntries(fixture.ctx, "zh-CN", shared.NewPageQuery(1, 20)); err != nil || total != 2 || len(entries) != 2 {
		t.Fatalf("sitemap entries=%#v total=%d err=%v", entries, total, err)
	}
	if err := fixture.db.Create(&modelCMS.ArticleTag{ArticleID: article.ID, TagID: tag.ID}).Error; err == nil {
		t.Fatal("expected duplicate article tag constraint")
	}
	if err := fixture.db.Model(&modelCMS.Category{}).Where("id = ?", category.ID).Update("is_enabled", false).Error; err != nil {
		t.Fatalf("disable category: %v", err)
	}
	if categories, err := fixture.repo.ListPublicCategoryTreeItems(fixture.ctx, "zh-CN"); err != nil || len(categories) != 0 {
		t.Fatalf("public categories=%#v err=%v", categories, err)
	}
}
