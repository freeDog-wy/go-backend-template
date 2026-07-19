package cms

import (
	"context"
	"time"

	domainCMS "github.com/freeDog-wy/go-backend-template/internal/domain/cms"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
	modelCMS "github.com/freeDog-wy/go-backend-template/internal/model/cms"
	"gorm.io/gorm/clause"
)

func (r *Repository) CreateCategory(ctx context.Context, category *domainCMS.Category, translation *domainCMS.CategoryTranslation) error {
	model := modelCMS.Category{ParentID: category.ParentID, SortOrder: category.SortOrder, IsEnabled: category.Enabled}
	if err := r.conn(ctx).Create(&model).Error; err != nil {
		return err
	}
	category.ID, category.CreatedAt, category.UpdatedAt = model.ID, model.CreatedAt, model.UpdatedAt
	return r.conn(ctx).Create(&modelCMS.CategoryTranslation{CategoryID: model.ID, Locale: translation.Locale, Name: translation.Name, Slug: translation.Slug, Description: translation.Description, SEOTitle: translation.SEOTitle, SEODescription: translation.SEODescription}).Error
}

func (r *Repository) UpsertCategoryTranslation(ctx context.Context, translation *domainCMS.CategoryTranslation) error {
	model := modelCMS.CategoryTranslation{CategoryID: translation.CategoryID, Locale: translation.Locale, Name: translation.Name, Slug: translation.Slug, Description: translation.Description, SEOTitle: translation.SEOTitle, SEODescription: translation.SEODescription}
	return r.conn(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "category_id"}, {Name: "locale"}}, DoUpdates: clause.Assignments(map[string]any{"name": model.Name, "slug": model.Slug, "description": model.Description, "seo_title": model.SEOTitle, "seo_description": model.SEODescription, "updated_at": time.Now()})}).Create(&model).Error
}

func (r *Repository) FindCategoryTranslation(ctx context.Context, categoryID uint, locale string) (*domainCMS.CategoryTranslation, error) {
	var model modelCMS.CategoryTranslation
	if err := r.conn(ctx).Where("category_id = ? AND locale = ?", categoryID, locale).First(&model).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &domainCMS.CategoryTranslation{CategoryID: model.CategoryID, Locale: model.Locale, Name: model.Name, Slug: model.Slug, Description: model.Description, SEOTitle: model.SEOTitle, SEODescription: model.SEODescription}, nil
}

func (r *Repository) FindCategory(ctx context.Context, id uint) (*domainCMS.Category, error) {
	var model modelCMS.Category
	if err := r.conn(ctx).First(&model, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return categoryEntity(model), nil
}

func (r *Repository) IsCategoryDescendant(ctx context.Context, ancestorID, candidateID uint) (bool, error) {
	var count int64
	err := r.conn(ctx).Raw(`WITH RECURSIVE descendants AS (
 SELECT id FROM categories WHERE parent_id = ?
 UNION ALL SELECT c.id FROM categories c JOIN descendants d ON c.parent_id = d.id
) SELECT COUNT(*) FROM descendants WHERE id = ?`, ancestorID, candidateID).Scan(&count).Error
	return count > 0, err
}

func (r *Repository) MoveCategory(ctx context.Context, id uint, parentID *uint, sortOrder int) error {
	result := r.conn(ctx).Model(&modelCMS.Category{}).Where("id = ?", id).Updates(map[string]any{"parent_id": parentID, "sort_order": sortOrder})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

func (r *Repository) UpdateCategory(ctx context.Context, id uint, enabled bool, sortOrder int) error {
	result := r.conn(ctx).Model(&modelCMS.Category{}).Where("id = ?", id).Updates(map[string]any{"is_enabled": enabled, "sort_order": sortOrder})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

func (r *Repository) ListCategories(ctx context.Context) ([]*domainCMS.Category, error) {
	var models []modelCMS.Category
	if err := r.conn(ctx).Order("parent_id NULLS FIRST, sort_order, id").Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*domainCMS.Category, 0, len(models))
	for _, model := range models {
		result = append(result, categoryEntity(model))
	}
	return result, nil
}

func (r *Repository) ListCategoryTreeItems(ctx context.Context, locale string) ([]*domainCMS.CategoryTreeItem, error) {
	return r.listCategoryTreeItems(ctx, locale, false)
}

func (r *Repository) ListPublicCategoryTreeItems(ctx context.Context, locale string) ([]*domainCMS.CategoryTreeItem, error) {
	return r.listCategoryTreeItems(ctx, locale, true)
}

func (r *Repository) listCategoryTreeItems(ctx context.Context, locale string, enabledOnly bool) ([]*domainCMS.CategoryTreeItem, error) {
	type row struct {
		CategoryID     uint
		ParentID       *uint
		SortOrder      int
		IsEnabled      bool
		Name           string
		Slug           string
		Description    string
		SEOTitle       string
		SEODescription string
	}
	var rows []row
	db := r.conn(ctx).Table("categories").Select("categories.id AS category_id, categories.parent_id, categories.sort_order, categories.is_enabled, category_translations.name, category_translations.slug, category_translations.description, category_translations.seo_title, category_translations.seo_description").Joins("JOIN category_translations ON category_translations.category_id = categories.id").Where("category_translations.locale = ?", locale)
	if enabledOnly {
		db = db.Where("categories.is_enabled")
	}
	if err := db.Order("categories.parent_id NULLS FIRST, categories.sort_order, categories.id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]*domainCMS.CategoryTreeItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, &domainCMS.CategoryTreeItem{Category: domainCMS.Category{ID: row.CategoryID, ParentID: row.ParentID, SortOrder: row.SortOrder, Enabled: row.IsEnabled}, CategoryTranslation: domainCMS.CategoryTranslation{CategoryID: row.CategoryID, Locale: locale, Name: row.Name, Slug: row.Slug, Description: row.Description, SEOTitle: row.SEOTitle, SEODescription: row.SEODescription}})
	}
	return items, nil
}

func categoryEntity(model modelCMS.Category) *domainCMS.Category {
	return &domainCMS.Category{ID: model.ID, ParentID: model.ParentID, SortOrder: model.SortOrder, Enabled: model.IsEnabled, CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt}
}
