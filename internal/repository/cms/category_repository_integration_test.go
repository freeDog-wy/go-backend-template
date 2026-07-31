//go:build integration

package cms

import (
	"testing"

	domainCMS "github.com/freeDog-wy/go-backend-template/internal/domain/cms"
)

func TestListCategoryTreeItemsKeepsUntranslatedCategories(t *testing.T) {
	fixture := newCMSIntegrationFixture(t)
	if err := fixture.repo.CreateLocale(fixture.ctx, &domainCMS.Locale{Code: "en-US", Name: "English", IsEnabled: true, SortOrder: 1}); err != nil {
		t.Fatal(err)
	}
	root := fixture.createCategory(t, "root")
	child := &domainCMS.Category{ParentID: &root.ID, Enabled: true}
	childTranslation := &domainCMS.CategoryTranslation{Locale: "zh-CN", Name: "child", Slug: "child"}
	if err := fixture.repo.CreateCategory(fixture.ctx, child, childTranslation); err != nil {
		t.Fatal(err)
	}
	if err := fixture.repo.UpsertCategoryTranslation(fixture.ctx, &domainCMS.CategoryTranslation{CategoryID: root.ID, Locale: "en-US", Name: "Root", Slug: "root"}); err != nil {
		t.Fatal(err)
	}

	items, err := fixture.repo.ListCategoryTreeItems(fixture.ctx, "en-US")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %#v", items)
	}
	if !items[0].HasTranslation || items[0].Name != "Root" || items[1].HasTranslation || items[1].ID != child.ID || items[1].ParentID == nil || *items[1].ParentID != root.ID {
		t.Fatalf("items = %#v", items)
	}
}
