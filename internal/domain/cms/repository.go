package cms

import (
	"context"
	"time"

	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
)

// Repository 定义 CMS 持久化契约。
//
// 写操作由 Usecase 的事务边界保护，Repository 不自行开启脱离 context 的事务。查询
// 方法返回的内容范围由方法名区分：Public 前缀只服务公开内容，IncludingDeleted 明确
// 包含软删除记录；调用方不得混用它们绕过发布或删除规则。
type Repository interface {
	LocaleEnabled(ctx context.Context, code string) (bool, error)
	ListLocales(ctx context.Context) ([]*Locale, error)
	FindLocale(ctx context.Context, code string) (*Locale, error)
	CreateLocale(ctx context.Context, locale *Locale) error
	UpdateLocale(ctx context.Context, locale *Locale) error
	// SetDefaultLocale 将一个已启用 locale 设为唯一默认 locale。
	SetDefaultLocale(ctx context.Context, code string) error
	CountEnabledLocales(ctx context.Context) (int64, error)
	CreateTag(ctx context.Context, tag *Tag, translation *TagTranslation) error
	FindTag(ctx context.Context, id uint) (*Tag, error)
	FindTagTranslation(ctx context.Context, tagID uint, locale string) (*TagTranslation, error)
	UpsertTagTranslation(ctx context.Context, translation *TagTranslation) error
	ListTags(ctx context.Context, locale string, page shared.PageQuery) ([]*TagListItem, int64, error)
	CreateCategory(ctx context.Context, category *Category, translation *CategoryTranslation) error
	UpsertCategoryTranslation(ctx context.Context, translation *CategoryTranslation) error
	FindCategoryTranslation(ctx context.Context, categoryID uint, locale string) (*CategoryTranslation, error)
	FindCategory(ctx context.Context, id uint) (*Category, error)
	IsCategoryDescendant(ctx context.Context, ancestorID, candidateID uint) (bool, error)
	MoveCategory(ctx context.Context, id uint, parentID *uint, sortOrder int) error
	UpdateCategory(ctx context.Context, id uint, enabled bool, sortOrder int) error
	ListCategories(ctx context.Context) ([]*Category, error)
	ListCategoryTreeItems(ctx context.Context, locale string) ([]*CategoryTreeItem, error)
	CreateArticle(ctx context.Context, article *Article, translation *ArticleTranslation) error
	FindArticle(ctx context.Context, id uint) (*Article, error)
	SetArticleCover(ctx context.Context, articleID uint, mediaID *uint) error
	FindArticleIncludingDeleted(ctx context.Context, id uint) (*Article, error)
	SoftDeleteArticle(ctx context.Context, id uint, deletedAt time.Time) error
	RestoreArticle(ctx context.Context, id uint) error
	CreateArticleTranslation(ctx context.Context, translation *ArticleTranslation) error
	FindArticleTranslation(ctx context.Context, articleID uint, locale string) (*ArticleTranslation, error)
	// RedirectSourceExists 用于在写入公开 slug 前检查路径和既有重定向冲突。
	RedirectSourceExists(ctx context.Context, locale, sourcePath string) (bool, error)
	// SaveURLRedirect 持久化由 slug 变更产生的路径重定向；应与新 slug 写入处于同一事务。
	SaveURLRedirect(ctx context.Context, redirect *URLRedirect) error
	FindURLRedirect(ctx context.Context, locale, sourcePath string) (*URLRedirect, error)
	ListURLRedirects(ctx context.Context, locale string, page shared.PageQuery) ([]URLRedirect, int64, error)
	ListArticleCategories(ctx context.Context, articleID uint) ([]ArticleCategory, error)
	ListArticleTags(ctx context.Context, articleID uint, locale string) ([]*TagListItem, error)
	// ReplaceArticleTags 以完整输入替换文章标签关系，而非增量追加。
	ReplaceArticleTags(ctx context.Context, articleID uint, tagIDs []uint) error
	SaveArticleTranslation(ctx context.Context, translation *ArticleTranslation) error
	// ReplaceArticleCategories 以完整输入替换分类关系，并要求主分类属于输入集合。
	ReplaceArticleCategories(ctx context.Context, articleID uint, categoryIDs []uint, primaryCategoryID *uint) error
	ListArticleTranslations(ctx context.Context, locale string, status TranslationStatus, includeDeleted bool, page shared.PageQuery) ([]*ArticleListItem, int64, error)
	// 以下 Public 方法仅返回满足公开可见条件的内容，不能作为后台管理读取的替代。
	FindPublicArticle(ctx context.Context, locale, slug string) (*PublicArticle, error)
	ListPublishedArticleLocales(ctx context.Context, articleID uint) ([]PublishedLocale, error)
	ListPublicArticleBreadcrumbs(ctx context.Context, articleID uint, locale string) ([]CategoryTreeItem, error)
	ListPublicSitemapEntries(ctx context.Context, locale string, page shared.PageQuery) ([]SitemapEntry, int64, error)
	ListPublicCategoryTreeItems(ctx context.Context, locale string) ([]*CategoryTreeItem, error)
	PublicCategoryExists(ctx context.Context, locale, slug string) (bool, error)
	ListPublicArticles(ctx context.Context, locale string, categorySlug *string, page shared.PageQuery) ([]*PublicArticleListItem, int64, error)
	PublicTagExists(ctx context.Context, locale, slug string) (bool, error)
	ListPublicTags(ctx context.Context, locale string, page shared.PageQuery) ([]*TagListItem, int64, error)
	ListPublicTagArticles(ctx context.Context, locale, tagSlug string, page shared.PageQuery) ([]*PublicArticleListItem, int64, error)
}

// The following focused ports allow use cases to depend only on the CMS data
// capabilities they actually consume. Repository remains temporarily for
// compatibility while callers migrate to Repositories.
type LocaleRepository interface {
	LocaleEnabled(context.Context, string) (bool, error)
	ListLocales(context.Context) ([]*Locale, error)
	FindLocale(context.Context, string) (*Locale, error)
	CreateLocale(context.Context, *Locale) error
	UpdateLocale(context.Context, *Locale) error
	SetDefaultLocale(context.Context, string) error
	CountEnabledLocales(context.Context) (int64, error)
}

type TagRepository interface {
	CreateTag(context.Context, *Tag, *TagTranslation) error
	FindTag(context.Context, uint) (*Tag, error)
	FindTagTranslation(context.Context, uint, string) (*TagTranslation, error)
	UpsertTagTranslation(context.Context, *TagTranslation) error
	ListTags(context.Context, string, shared.PageQuery) ([]*TagListItem, int64, error)
}

type CategoryRepository interface {
	CreateCategory(context.Context, *Category, *CategoryTranslation) error
	UpsertCategoryTranslation(context.Context, *CategoryTranslation) error
	FindCategoryTranslation(context.Context, uint, string) (*CategoryTranslation, error)
	FindCategory(context.Context, uint) (*Category, error)
	IsCategoryDescendant(context.Context, uint, uint) (bool, error)
	MoveCategory(context.Context, uint, *uint, int) error
	UpdateCategory(context.Context, uint, bool, int) error
	ListCategories(context.Context) ([]*Category, error)
	ListCategoryTreeItems(context.Context, string) ([]*CategoryTreeItem, error)
	ListPublicCategoryTreeItems(context.Context, string) ([]*CategoryTreeItem, error)
}

type ArticleRepository interface {
	CreateArticle(context.Context, *Article, *ArticleTranslation) error
	FindArticle(context.Context, uint) (*Article, error)
	SetArticleCover(context.Context, uint, *uint) error
	FindArticleIncludingDeleted(context.Context, uint) (*Article, error)
	SoftDeleteArticle(context.Context, uint, time.Time) error
	RestoreArticle(context.Context, uint) error
	CreateArticleTranslation(context.Context, *ArticleTranslation) error
	FindArticleTranslation(context.Context, uint, string) (*ArticleTranslation, error)
	SaveArticleTranslation(context.Context, *ArticleTranslation) error
	ListArticleTranslations(context.Context, string, TranslationStatus, bool, shared.PageQuery) ([]*ArticleListItem, int64, error)
}

type ArticleRelationRepository interface {
	ListArticleCategories(context.Context, uint) ([]ArticleCategory, error)
	ListArticleTags(context.Context, uint, string) ([]*TagListItem, error)
	ReplaceArticleTags(context.Context, uint, []uint) error
	ReplaceArticleCategories(context.Context, uint, []uint, *uint) error
}

type RedirectRepository interface {
	RedirectSourceExists(context.Context, string, string) (bool, error)
	SaveURLRedirect(context.Context, *URLRedirect) error
	FindURLRedirect(context.Context, string, string) (*URLRedirect, error)
	ListURLRedirects(context.Context, string, shared.PageQuery) ([]URLRedirect, int64, error)
}

type PublicContentRepository interface {
	FindPublicArticle(context.Context, string, string) (*PublicArticle, error)
	ListPublishedArticleLocales(context.Context, uint) ([]PublishedLocale, error)
	ListPublicArticleBreadcrumbs(context.Context, uint, string) ([]CategoryTreeItem, error)
	ListPublicSitemapEntries(context.Context, string, shared.PageQuery) ([]SitemapEntry, int64, error)
	PublicCategoryExists(context.Context, string, string) (bool, error)
	ListPublicArticles(context.Context, string, *string, shared.PageQuery) ([]*PublicArticleListItem, int64, error)
	PublicTagExists(context.Context, string, string) (bool, error)
	ListPublicTags(context.Context, string, shared.PageQuery) ([]*TagListItem, int64, error)
	ListPublicTagArticles(context.Context, string, string, shared.PageQuery) ([]*PublicArticleListItem, int64, error)
}
