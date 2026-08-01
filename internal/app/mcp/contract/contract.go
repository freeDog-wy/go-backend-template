package contract

import (
	"context"
	"encoding/json"

	"github.com/freeDog-wy/go-backend-template/internal/app/pkg/cmsclient"
)

type ArticleInput = cmsclient.ArticleInput
type CategoryInput = cmsclient.CategoryInput
type CategoryStateInput = cmsclient.CategoryStateInput
type CategoryMoveInput = cmsclient.CategoryMoveInput
type CategoryTranslationInput = cmsclient.CategoryTranslationInput
type RenameInput = cmsclient.RenameInput
type TagInput = cmsclient.TagInput
type TagTranslationInput = cmsclient.TagTranslationInput
type TagStateInput = cmsclient.TagStateInput
type LocaleCreateInput = cmsclient.LocaleCreateInput
type LocaleUpdateInput = cmsclient.LocaleUpdateInput
type ArticleListOptions = cmsclient.ArticleListOptions
type MediaUploadRequestInput = cmsclient.MediaUploadRequestInput
type MediaTranslationInput = cmsclient.MediaTranslationInput

type SiteReader interface {
	Health(context.Context) (json.RawMessage, error)
}

type LocaleService interface {
	Locales(context.Context) (json.RawMessage, error)
	CreateLocale(context.Context, LocaleCreateInput) (json.RawMessage, error)
	UpdateLocale(context.Context, string, LocaleUpdateInput) (json.RawMessage, error)
}

type ArticleService interface {
	Articles(context.Context, string, string, int, int, ArticleListOptions) (json.RawMessage, error)
	ArticleTranslation(context.Context, uint, string) (json.RawMessage, error)
	CreateArticleDraft(context.Context, ArticleInput) (json.RawMessage, error)
	CreateArticleTranslation(context.Context, uint, ArticleInput) (json.RawMessage, error)
	UpdateArticleTranslation(context.Context, uint, string, ArticleInput) (json.RawMessage, error)
	ReplaceArticleCategories(context.Context, uint, []uint, *uint) (json.RawMessage, error)
	ReplaceArticleTags(context.Context, uint, []uint) (json.RawMessage, error)
	PreviewPublish(context.Context, uint, string) (json.RawMessage, error)
	PublishArticleTranslation(context.Context, uint, string) (json.RawMessage, error)
	ArchiveArticleTranslation(context.Context, uint, string) (json.RawMessage, error)
	DeleteArticle(context.Context, uint) (json.RawMessage, error)
	RestoreArticle(context.Context, uint) (json.RawMessage, error)
	SetArticleCover(context.Context, uint, *uint) (json.RawMessage, error)
	PreviewMarkdown(context.Context, string) (json.RawMessage, error)
}

type CategoryService interface {
	Categories(context.Context, string) (json.RawMessage, error)
	CreateCategory(context.Context, CategoryInput) (json.RawMessage, error)
	UpdateCategory(context.Context, uint, CategoryStateInput) (json.RawMessage, error)
	MoveCategory(context.Context, uint, CategoryMoveInput) (json.RawMessage, error)
	RenameCategory(context.Context, uint, string, RenameInput) (json.RawMessage, error)
	DeleteCategory(context.Context, uint) (json.RawMessage, error)
	UpsertCategoryTranslation(context.Context, uint, string, CategoryTranslationInput) (json.RawMessage, error)
}

type TagService interface {
	Tags(context.Context, string, int, int) (json.RawMessage, error)
	CreateTag(context.Context, TagInput) (json.RawMessage, error)
	UpdateTag(context.Context, uint, TagStateInput) (json.RawMessage, error)
	RenameTag(context.Context, uint, string, RenameInput) (json.RawMessage, error)
	UpsertTagTranslation(context.Context, uint, string, TagTranslationInput) (json.RawMessage, error)
}

type MediaService interface {
	Media(context.Context, int, int) (json.RawMessage, error)
	RequestMediaUpload(context.Context, MediaUploadRequestInput) (json.RawMessage, error)
	CompleteMediaUpload(context.Context, uint) (json.RawMessage, error)
	UpsertMediaTranslation(context.Context, uint, string, MediaTranslationInput) (json.RawMessage, error)
}

type APIError = cmsclient.APIError

// WithWriteOperation binds an MCP write intent to its duplicate-write guard key.
func WithWriteOperation(ctx context.Context, operationID string) context.Context {
	return cmsclient.WithWriteOperation(ctx, operationID)
}

func WriteOperationID(ctx context.Context) string {
	return cmsclient.WriteOperationID(ctx)
}
