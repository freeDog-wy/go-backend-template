package cms

import (
	"context"

	domainCMS "github.com/freeDog-wy/go-backend-template/internal/domain/cms"
	modelCMS "github.com/freeDog-wy/go-backend-template/internal/model/cms"
)

func (r *Repository) ListArticleCategories(ctx context.Context, articleID uint) ([]domainCMS.ArticleCategory, error) {
	var models []modelCMS.ArticleCategory
	if err := r.conn(ctx).Where("article_id = ?", articleID).Order("is_primary DESC, category_id").Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]domainCMS.ArticleCategory, 0, len(models))
	for _, model := range models {
		result = append(result, domainCMS.ArticleCategory{CategoryID: model.CategoryID, IsPrimary: model.IsPrimary})
	}
	return result, nil
}

func (r *Repository) ListArticleTags(ctx context.Context, articleID uint, locale string) ([]*domainCMS.TagListItem, error) {
	type row struct {
		TagID      uint
		IsEnabled  bool
		Name, Slug string
	}
	var rows []row
	err := r.conn(ctx).Table("article_tags").Joins("JOIN tags ON tags.id = article_tags.tag_id").Joins("LEFT JOIN tag_translations ON tag_translations.tag_id = tags.id AND tag_translations.locale = ?", locale).Where("article_tags.article_id = ?", articleID).Order("tag_translations.name NULLS LAST, tags.id").Select("tags.id AS tag_id, tags.is_enabled, tag_translations.name, tag_translations.slug").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]*domainCMS.TagListItem, 0, len(rows))
	for _, row := range rows {
		result = append(result, &domainCMS.TagListItem{Tag: domainCMS.Tag{ID: row.TagID, Enabled: row.IsEnabled}, TagTranslation: domainCMS.TagTranslation{TagID: row.TagID, Locale: locale, Name: row.Name, Slug: row.Slug}})
	}
	return result, nil
}

// ListPublicArticleTags returns only enabled tags with a translation in the
// requested locale. The result is grouped to populate a page of articles in
// one query rather than issuing a query for each article.
func (r *Repository) ListPublicArticleTags(ctx context.Context, articleIDs []uint, locale string) (map[uint][]*domainCMS.TagListItem, error) {
	result := make(map[uint][]*domainCMS.TagListItem)
	if len(articleIDs) == 0 {
		return result, nil
	}

	type row struct {
		ArticleID uint
		TagID     uint
		Name      string
		Slug      string
	}
	var rows []row
	err := r.conn(ctx).Table("article_tags").
		Joins("JOIN tags ON tags.id = article_tags.tag_id AND tags.is_enabled").
		Joins("JOIN tag_translations ON tag_translations.tag_id = tags.id AND tag_translations.locale = ?", locale).
		Where("article_tags.article_id IN ?", articleIDs).
		Select("article_tags.article_id, tags.id AS tag_id, tag_translations.name, tag_translations.slug").
		Order("article_tags.article_id, tag_translations.name, tags.id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.ArticleID] = append(result[row.ArticleID], &domainCMS.TagListItem{
			Tag:            domainCMS.Tag{ID: row.TagID, Enabled: true},
			TagTranslation: domainCMS.TagTranslation{TagID: row.TagID, Locale: locale, Name: row.Name, Slug: row.Slug},
		})
	}
	return result, nil
}

func (r *Repository) ReplaceArticleTags(ctx context.Context, articleID uint, tagIDs []uint) error {
	db := r.conn(ctx)
	if err := db.Where("article_id = ?", articleID).Delete(&modelCMS.ArticleTag{}).Error; err != nil {
		return err
	}
	if len(tagIDs) == 0 {
		return nil
	}
	records := make([]modelCMS.ArticleTag, 0, len(tagIDs))
	for _, tagID := range tagIDs {
		records = append(records, modelCMS.ArticleTag{ArticleID: articleID, TagID: tagID})
	}
	return db.Create(&records).Error
}

func (r *Repository) ReplaceArticleCategories(ctx context.Context, articleID uint, categoryIDs []uint, primaryCategoryID *uint) error {
	db := r.conn(ctx)
	if err := db.Where("article_id = ?", articleID).Delete(&modelCMS.ArticleCategory{}).Error; err != nil {
		return err
	}
	if len(categoryIDs) == 0 {
		return nil
	}
	records := make([]modelCMS.ArticleCategory, 0, len(categoryIDs))
	for _, categoryID := range categoryIDs {
		records = append(records, modelCMS.ArticleCategory{ArticleID: articleID, CategoryID: categoryID, IsPrimary: primaryCategoryID != nil && *primaryCategoryID == categoryID})
	}
	return db.Create(&records).Error
}
