package cms

import (
	"context"
	"time"

	domainCMS "github.com/freeDog-wy/go-backend-template/internal/domain/cms"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
	modelCMS "github.com/freeDog-wy/go-backend-template/internal/model/cms"
)

func (r *Repository) CreateArticle(ctx context.Context, article *domainCMS.Article, tr *domainCMS.ArticleTranslation) error {
	m := modelCMS.Article{AuthorUserID: article.AuthorUserID}
	if err := r.conn(ctx).Create(&m).Error; err != nil {
		return err
	}
	article.ID, article.CreatedAt, article.UpdatedAt = m.ID, m.CreatedAt, m.UpdatedAt
	tm := translationModel(m.ID, tr)
	if err := r.conn(ctx).Create(&tm).Error; err != nil {
		return err
	}
	tr.ID, tr.ArticleID, tr.CreatedAt, tr.UpdatedAt = tm.ID, m.ID, tm.CreatedAt, tm.UpdatedAt
	return nil
}

func (r *Repository) FindArticle(ctx context.Context, id uint) (*domainCMS.Article, error) {
	return r.findArticle(ctx, id, false)
}
func (r *Repository) SetArticleCover(ctx context.Context, articleID uint, mediaID *uint) error {
	res := r.conn(ctx).Model(&modelCMS.Article{}).Where("id = ? AND deleted_at IS NULL", articleID).Update("cover_media_id", mediaID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}
func (r *Repository) FindArticleIncludingDeleted(ctx context.Context, id uint) (*domainCMS.Article, error) {
	return r.findArticle(ctx, id, true)
}
func (r *Repository) findArticle(ctx context.Context, id uint, includeDeleted bool) (*domainCMS.Article, error) {
	var m modelCMS.Article
	db := r.conn(ctx)
	if !includeDeleted {
		db = db.Where("deleted_at IS NULL")
	}
	if err := db.First(&m, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &domainCMS.Article{ID: m.ID, AuthorUserID: m.AuthorUserID, CoverMediaID: m.CoverMediaID, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt, DeletedAt: m.DeletedAt}, nil
}

func (r *Repository) SoftDeleteArticle(ctx context.Context, id uint, deletedAt time.Time) error {
	result := r.conn(ctx).Model(&modelCMS.Article{}).Where("id = ? AND deleted_at IS NULL", id).Updates(map[string]any{"deleted_at": deletedAt})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}
func (r *Repository) RestoreArticle(ctx context.Context, id uint) error {
	result := r.conn(ctx).Model(&modelCMS.Article{}).Where("id = ? AND deleted_at IS NOT NULL", id).Update("deleted_at", nil)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

func (r *Repository) FindArticleTranslation(ctx context.Context, articleID uint, locale string) (*domainCMS.ArticleTranslation, error) {
	var m modelCMS.ArticleTranslation
	if err := r.conn(ctx).Where("article_id = ? AND locale = ?", articleID, locale).First(&m).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return translationEntity(m), nil
}

func (r *Repository) CreateArticleTranslation(ctx context.Context, tr *domainCMS.ArticleTranslation) error {
	m := translationModel(tr.ArticleID, tr)
	if err := r.conn(ctx).Create(&m).Error; err != nil {
		return err
	}
	tr.ID, tr.CreatedAt, tr.UpdatedAt = m.ID, m.CreatedAt, m.UpdatedAt
	return nil
}

func (r *Repository) SaveArticleTranslation(ctx context.Context, tr *domainCMS.ArticleTranslation) error {
	m := translationModel(tr.ArticleID, tr)
	result := r.conn(ctx).Model(&modelCMS.ArticleTranslation{}).Where("id = ?", tr.ID).Updates(map[string]any{"title": m.Title, "slug": m.Slug, "summary": m.Summary, "content": m.Content, "content_format": m.ContentFormat, "status": m.Status, "published_at": m.PublishedAt, "seo_title": m.SEOTitle, "seo_description": m.SEODescription, "canonical_url": m.CanonicalURL})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

func (r *Repository) ListArticleTranslations(ctx context.Context, locale string, status domainCMS.TranslationStatus, includeDeleted bool, page shared.PageQuery) ([]*domainCMS.ArticleListItem, int64, error) {
	db := r.conn(ctx).Table("article_translations").Joins("JOIN articles ON articles.id = article_translations.article_id").Where("article_translations.locale = ?", locale)
	if status != "" {
		db = db.Where("article_translations.status = ?", status)
	}
	if !includeDeleted {
		db = db.Where("articles.deleted_at IS NULL")
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	type row struct {
		ArticleID                                                                                    uint
		AuthorUserID                                                                                 uint
		CoverMediaID                                                                                 *uint
		ArticleCreatedAt, ArticleUpdatedAt                                                           time.Time
		TranslationID                                                                                uint
		Title, Slug, Summary, Content, ContentFormat, Status, SEOTitle, SEODescription, CanonicalURL string
		PublishedAt                                                                                  *time.Time
		TranslationCreatedAt, TranslationUpdatedAt                                                   time.Time
	}
	var rows []row
	err := db.Select("articles.id AS article_id, articles.author_user_id, articles.cover_media_id, articles.created_at AS article_created_at, articles.updated_at AS article_updated_at, article_translations.id AS translation_id, article_translations.title, article_translations.slug, article_translations.summary, article_translations.content, article_translations.content_format, article_translations.status, article_translations.published_at, article_translations.seo_title, article_translations.seo_description, article_translations.canonical_url, article_translations.created_at AS translation_created_at, article_translations.updated_at AS translation_updated_at").Order("article_translations.updated_at DESC, article_translations.id DESC").Limit(page.PerPage).Offset(page.Offset()).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	items := make([]*domainCMS.ArticleListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, &domainCMS.ArticleListItem{Article: domainCMS.Article{ID: row.ArticleID, AuthorUserID: row.AuthorUserID, CoverMediaID: row.CoverMediaID, CreatedAt: row.ArticleCreatedAt, UpdatedAt: row.ArticleUpdatedAt}, ArticleTranslation: domainCMS.ArticleTranslation{ID: row.TranslationID, ArticleID: row.ArticleID, Locale: locale, Title: row.Title, Slug: row.Slug, Summary: row.Summary, Content: row.Content, ContentFormat: row.ContentFormat, Status: domainCMS.TranslationStatus(row.Status), PublishedAt: row.PublishedAt, SEOTitle: row.SEOTitle, SEODescription: row.SEODescription, CanonicalURL: row.CanonicalURL, CreatedAt: row.TranslationCreatedAt, UpdatedAt: row.TranslationUpdatedAt}})
	}
	return items, total, nil
}

func (r *Repository) FindPublicArticle(ctx context.Context, locale, slug string) (*domainCMS.PublicArticle, error) {
	var article modelCMS.Article
	err := r.conn(ctx).Table("articles").Select("articles.*").Joins("JOIN article_translations ON article_translations.article_id = articles.id").Where("articles.deleted_at IS NULL AND article_translations.locale = ? AND article_translations.slug = ? AND article_translations.status = 'published' AND article_translations.published_at <= NOW()", locale, slug).First(&article).Error
	if err != nil {
		return nil, mapNotFound(err)
	}
	tr, err := r.FindArticleTranslation(ctx, article.ID, locale)
	if err != nil {
		return nil, err
	}
	return &domainCMS.PublicArticle{Article: domainCMS.Article{ID: article.ID, AuthorUserID: article.AuthorUserID, CoverMediaID: article.CoverMediaID, CreatedAt: article.CreatedAt, UpdatedAt: article.UpdatedAt, DeletedAt: article.DeletedAt}, ArticleTranslation: *tr}, nil
}

func (r *Repository) ListPublishedArticleLocales(ctx context.Context, articleID uint) ([]domainCMS.PublishedLocale, error) {
	type row struct{ Locale, Slug string }
	var rows []row
	err := r.conn(ctx).Table("article_translations").Joins("JOIN locales ON locales.code = article_translations.locale").Where("article_translations.article_id = ? AND article_translations.status = 'published' AND article_translations.published_at <= NOW() AND locales.is_enabled", articleID).Order("locales.sort_order, article_translations.locale").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]domainCMS.PublishedLocale, 0, len(rows))
	for _, row := range rows {
		result = append(result, domainCMS.PublishedLocale{Locale: row.Locale, Slug: row.Slug})
	}
	return result, nil
}

func (r *Repository) ListPublicArticleBreadcrumbs(ctx context.Context, articleID uint, locale string) ([]domainCMS.CategoryTreeItem, error) {
	type row struct {
		CategoryID              uint
		ParentID                *uint
		SortOrder               int
		Name, Slug, Description string
	}
	var rows []row
	err := r.conn(ctx).Raw(`WITH RECURSIVE path AS (
  SELECT c.id, c.parent_id, c.sort_order, 1 AS depth
  FROM article_categories ac JOIN categories c ON c.id = ac.category_id
  WHERE ac.article_id = ? AND ac.is_primary AND c.is_enabled
  UNION ALL
  SELECT parent.id, parent.parent_id, parent.sort_order, path.depth + 1
  FROM categories parent JOIN path ON path.parent_id = parent.id
  WHERE parent.is_enabled
)
SELECT path.id AS category_id, path.parent_id, path.sort_order, ct.name, ct.slug, ct.description
FROM path JOIN category_translations ct ON ct.category_id = path.id AND ct.locale = ?
ORDER BY path.depth DESC`, articleID, locale).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]domainCMS.CategoryTreeItem, 0, len(rows))
	for _, row := range rows {
		result = append(result, domainCMS.CategoryTreeItem{Category: domainCMS.Category{ID: row.CategoryID, ParentID: row.ParentID, SortOrder: row.SortOrder, Enabled: true}, CategoryTranslation: domainCMS.CategoryTranslation{CategoryID: row.CategoryID, Locale: locale, Name: row.Name, Slug: row.Slug, Description: row.Description}})
	}
	return result, nil
}

func (r *Repository) ListPublicSitemapEntries(ctx context.Context, locale string, page shared.PageQuery) ([]domainCMS.SitemapEntry, int64, error) {
	base := `
SELECT 'article' AS kind, article_translations.slug, article_translations.updated_at
FROM article_translations
JOIN articles ON articles.id = article_translations.article_id
JOIN locales ON locales.code = article_translations.locale
WHERE article_translations.locale = ? AND article_translations.status = 'published' AND article_translations.published_at <= NOW() AND articles.deleted_at IS NULL AND locales.is_enabled
UNION ALL
SELECT 'category' AS kind, category_translations.slug, category_translations.updated_at
FROM category_translations
JOIN categories ON categories.id = category_translations.category_id
JOIN locales ON locales.code = category_translations.locale
WHERE category_translations.locale = ? AND categories.is_enabled AND locales.is_enabled`
	var total int64
	if err := r.conn(ctx).Raw("SELECT COUNT(*) FROM ("+base+") AS sitemap_entries", locale, locale).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	type row struct {
		Kind, Slug string
		UpdatedAt  time.Time
	}
	var rows []row
	query := "SELECT kind, slug, updated_at FROM (" + base + ") AS sitemap_entries ORDER BY updated_at DESC, kind, slug LIMIT ? OFFSET ?"
	if err := r.conn(ctx).Raw(query, locale, locale, page.PerPage, page.Offset()).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	result := make([]domainCMS.SitemapEntry, 0, len(rows))
	for _, row := range rows {
		result = append(result, domainCMS.SitemapEntry{Kind: row.Kind, Slug: row.Slug, UpdatedAt: row.UpdatedAt})
	}
	return result, total, nil
}

func (r *Repository) PublicCategoryExists(ctx context.Context, locale, slug string) (bool, error) {
	var count int64
	err := r.conn(ctx).Table("categories").Joins("JOIN category_translations ON category_translations.category_id = categories.id").Where("categories.is_enabled AND category_translations.locale = ? AND category_translations.slug = ?", locale, slug).Count(&count).Error
	return count == 1, err
}

func (r *Repository) ListPublicArticles(ctx context.Context, locale string, categorySlug *string, page shared.PageQuery) ([]*domainCMS.PublicArticleListItem, int64, error) {
	db := r.conn(ctx).Table("article_translations").
		Joins("JOIN articles ON articles.id = article_translations.article_id").
		Joins("LEFT JOIN article_categories primary_ac ON primary_ac.article_id = articles.id AND primary_ac.is_primary").
		Joins("LEFT JOIN categories primary_c ON primary_c.id = primary_ac.category_id AND primary_c.is_enabled").
		Joins("LEFT JOIN category_translations primary_ct ON primary_ct.category_id = primary_c.id AND primary_ct.locale = article_translations.locale").
		Where("articles.deleted_at IS NULL AND article_translations.locale = ? AND article_translations.status = 'published' AND article_translations.published_at <= NOW()", locale)
	if categorySlug != nil {
		db = db.Joins("JOIN article_categories filter_ac ON filter_ac.article_id = articles.id").
			Joins("JOIN categories filter_c ON filter_c.id = filter_ac.category_id AND filter_c.is_enabled").
			Joins("JOIN category_translations filter_ct ON filter_ct.category_id = filter_c.id AND filter_ct.locale = article_translations.locale").
			Where("filter_ct.slug = ?", *categorySlug)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	type row struct {
		ArticleID                                uint
		CoverMediaID                             *uint
		Title, Slug, Summary, ContentFormat      string
		PublishedAt                              *time.Time
		UpdatedAt                                time.Time
		PrimaryCategoryID                        *uint
		PrimaryCategoryName, PrimaryCategorySlug string
	}
	var rows []row
	err := db.Select("articles.id AS article_id, articles.cover_media_id, article_translations.title, article_translations.slug, article_translations.summary, article_translations.content_format, article_translations.published_at, article_translations.updated_at, primary_c.id AS primary_category_id, primary_ct.name AS primary_category_name, primary_ct.slug AS primary_category_slug").Order("article_translations.published_at DESC, article_translations.id DESC").Limit(page.PerPage).Offset(page.Offset()).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	items := make([]*domainCMS.PublicArticleListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, &domainCMS.PublicArticleListItem{Article: domainCMS.Article{ID: row.ArticleID, CoverMediaID: row.CoverMediaID, UpdatedAt: row.UpdatedAt}, ArticleTranslation: domainCMS.ArticleTranslation{ArticleID: row.ArticleID, Locale: locale, Title: row.Title, Slug: row.Slug, Summary: row.Summary, ContentFormat: row.ContentFormat, PublishedAt: row.PublishedAt, Status: domainCMS.TranslationPublished, UpdatedAt: row.UpdatedAt}, PrimaryCategoryID: row.PrimaryCategoryID, PrimaryCategoryName: row.PrimaryCategoryName, PrimaryCategorySlug: row.PrimaryCategorySlug})
	}
	return items, total, nil
}

func (r *Repository) PublicTagExists(ctx context.Context, locale, slug string) (bool, error) {
	var count int64
	err := r.conn(ctx).Table("tag_translations").Joins("JOIN locales ON locales.code = tag_translations.locale").Where("tag_translations.locale = ? AND tag_translations.slug = ? AND locales.is_enabled", locale, slug).Count(&count).Error
	return count == 1, err
}
func (r *Repository) ListPublicTags(ctx context.Context, locale string, page shared.PageQuery) ([]*domainCMS.TagListItem, int64, error) {
	base := r.conn(ctx).Table("tag_translations").
		Joins("JOIN tags ON tags.id = tag_translations.tag_id").
		Joins("JOIN article_tags ON article_tags.tag_id = tags.id").
		Joins("JOIN articles ON articles.id = article_tags.article_id").
		Joins("JOIN article_translations ON article_translations.article_id = articles.id AND article_translations.locale = tag_translations.locale").
		Where("tag_translations.locale = ? AND articles.deleted_at IS NULL AND article_translations.status = ? AND article_translations.published_at <= CURRENT_TIMESTAMP", locale, domainCMS.TranslationPublished)
	var total int64
	if err := base.Distinct("tags.id").Count(&total).Error; err != nil {
		return nil, 0, err
	}
	type row struct {
		TagID      uint
		Name, Slug string
	}
	var rows []row
	if err := base.Select("tags.id AS tag_id, tag_translations.name, tag_translations.slug").Group("tags.id, tag_translations.name, tag_translations.slug").Order("tag_translations.name, tags.id").Limit(page.PerPage).Offset(page.Offset()).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	result := make([]*domainCMS.TagListItem, 0, len(rows))
	for _, row := range rows {
		result = append(result, &domainCMS.TagListItem{Tag: domainCMS.Tag{ID: row.TagID}, TagTranslation: domainCMS.TagTranslation{TagID: row.TagID, Locale: locale, Name: row.Name, Slug: row.Slug}})
	}
	return result, total, nil
}
func (r *Repository) ListPublicTagArticles(ctx context.Context, locale, tagSlug string, page shared.PageQuery) ([]*domainCMS.PublicArticleListItem, int64, error) {
	db := r.conn(ctx).Table("article_translations").Joins("JOIN articles ON articles.id = article_translations.article_id").Joins("LEFT JOIN article_categories primary_ac ON primary_ac.article_id = articles.id AND primary_ac.is_primary").Joins("LEFT JOIN categories primary_c ON primary_c.id = primary_ac.category_id AND primary_c.is_enabled").Joins("LEFT JOIN category_translations primary_ct ON primary_ct.category_id = primary_c.id AND primary_ct.locale = article_translations.locale").Joins("JOIN article_tags filter_at ON filter_at.article_id = articles.id").Joins("JOIN tag_translations filter_tt ON filter_tt.tag_id = filter_at.tag_id AND filter_tt.locale = article_translations.locale").Where("articles.deleted_at IS NULL AND article_translations.locale = ? AND article_translations.status = 'published' AND article_translations.published_at <= NOW() AND filter_tt.slug = ?", locale, tagSlug)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	type row struct {
		ArticleID                                uint
		CoverMediaID                             *uint
		Title, Slug, Summary, ContentFormat      string
		PublishedAt                              *time.Time
		UpdatedAt                                time.Time
		PrimaryCategoryID                        *uint
		PrimaryCategoryName, PrimaryCategorySlug string
	}
	var rows []row
	if err := db.Select("articles.id AS article_id, articles.cover_media_id, article_translations.title, article_translations.slug, article_translations.summary, article_translations.content_format, article_translations.published_at, article_translations.updated_at, primary_c.id AS primary_category_id, primary_ct.name AS primary_category_name, primary_ct.slug AS primary_category_slug").Order("article_translations.published_at DESC, article_translations.id DESC").Limit(page.PerPage).Offset(page.Offset()).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	result := make([]*domainCMS.PublicArticleListItem, 0, len(rows))
	for _, row := range rows {
		result = append(result, &domainCMS.PublicArticleListItem{Article: domainCMS.Article{ID: row.ArticleID, CoverMediaID: row.CoverMediaID, UpdatedAt: row.UpdatedAt}, ArticleTranslation: domainCMS.ArticleTranslation{ArticleID: row.ArticleID, Locale: locale, Title: row.Title, Slug: row.Slug, Summary: row.Summary, ContentFormat: row.ContentFormat, PublishedAt: row.PublishedAt, Status: domainCMS.TranslationPublished, UpdatedAt: row.UpdatedAt}, PrimaryCategoryID: row.PrimaryCategoryID, PrimaryCategoryName: row.PrimaryCategoryName, PrimaryCategorySlug: row.PrimaryCategorySlug})
	}
	return result, total, nil
}

func translationModel(articleID uint, tr *domainCMS.ArticleTranslation) modelCMS.ArticleTranslation {
	return modelCMS.ArticleTranslation{ID: tr.ID, ArticleID: articleID, Locale: tr.Locale, Title: tr.Title, Slug: tr.Slug, Summary: tr.Summary, Content: tr.Content, ContentFormat: tr.ContentFormat, Status: string(tr.Status), PublishedAt: tr.PublishedAt, SEOTitle: tr.SEOTitle, SEODescription: tr.SEODescription, CanonicalURL: tr.CanonicalURL}
}
func translationEntity(m modelCMS.ArticleTranslation) *domainCMS.ArticleTranslation {
	return &domainCMS.ArticleTranslation{ID: m.ID, ArticleID: m.ArticleID, Locale: m.Locale, Title: m.Title, Slug: m.Slug, Summary: m.Summary, Content: m.Content, ContentFormat: m.ContentFormat, Status: domainCMS.TranslationStatus(m.Status), PublishedAt: m.PublishedAt, SEOTitle: m.SEOTitle, SEODescription: m.SEODescription, CanonicalURL: m.CanonicalURL, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}
}
