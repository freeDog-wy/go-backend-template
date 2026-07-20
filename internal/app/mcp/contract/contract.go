package contract

import (
	"context"
	"encoding/json"
	"strings"
)

type ArticleInput struct {
	Locale         string `json:"locale"`
	Title          string `json:"title"`
	Slug           string `json:"slug"`
	Summary        string `json:"summary,omitempty"`
	Content        string `json:"content,omitempty"`
	ContentFormat  string `json:"content_format,omitempty"`
	SEOTitle       string `json:"seo_title,omitempty"`
	SEODescription string `json:"seo_description,omitempty"`
	CanonicalURL   string `json:"canonical_url,omitempty"`
}

type CategoryInput struct {
	ParentID       *uint  `json:"parent_id,omitempty"`
	SortOrder      int    `json:"sort_order"`
	Locale         string `json:"locale"`
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	Description    string `json:"description,omitempty"`
	SEOTitle       string `json:"seo_title,omitempty"`
	SEODescription string `json:"seo_description,omitempty"`
}

type CategoryStateInput struct {
	IsEnabled bool `json:"is_enabled"`
	SortOrder int  `json:"sort_order"`
}

type CategoryMoveInput struct {
	ParentID  *uint `json:"parent_id,omitempty"`
	SortOrder int   `json:"sort_order"`
}

type CategoryTranslationInput struct {
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	Description    string `json:"description,omitempty"`
	SEOTitle       string `json:"seo_title,omitempty"`
	SEODescription string `json:"seo_description,omitempty"`
}

type TagInput struct {
	Locale string `json:"locale"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
}

type TagTranslationInput struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type LocaleCreateInput struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	IsEnabled bool   `json:"is_enabled"`
	SortOrder int    `json:"sort_order"`
}

type LocaleUpdateInput struct {
	Name      string `json:"name"`
	IsEnabled bool   `json:"is_enabled"`
	SortOrder int    `json:"sort_order"`
	IsDefault bool   `json:"is_default"`
}

type SiteReader interface {
	Health(context.Context) (json.RawMessage, error)
}

type LocaleService interface {
	Locales(context.Context) (json.RawMessage, error)
	CreateLocale(context.Context, LocaleCreateInput) (json.RawMessage, error)
	UpdateLocale(context.Context, string, LocaleUpdateInput) (json.RawMessage, error)
}

type ArticleService interface {
	Articles(context.Context, string, string, int, int) (json.RawMessage, error)
	ArticleTranslation(context.Context, uint, string) (json.RawMessage, error)
	CreateArticleDraft(context.Context, ArticleInput) (json.RawMessage, error)
	CreateArticleTranslation(context.Context, uint, ArticleInput) (json.RawMessage, error)
	UpdateArticleTranslation(context.Context, uint, string, ArticleInput) (json.RawMessage, error)
	ReplaceArticleCategories(context.Context, uint, []uint, *uint) (json.RawMessage, error)
	ReplaceArticleTags(context.Context, uint, []uint) (json.RawMessage, error)
	PreviewPublish(context.Context, uint, string) (json.RawMessage, error)
	PublishArticleTranslation(context.Context, uint, string) (json.RawMessage, error)
	ArchiveArticleTranslation(context.Context, uint, string) (json.RawMessage, error)
	RestoreArticle(context.Context, uint) (json.RawMessage, error)
	SetArticleCover(context.Context, uint, *uint) (json.RawMessage, error)
}

type CategoryService interface {
	Categories(context.Context, string) (json.RawMessage, error)
	CreateCategory(context.Context, CategoryInput) (json.RawMessage, error)
	UpdateCategory(context.Context, uint, CategoryStateInput) (json.RawMessage, error)
	MoveCategory(context.Context, uint, CategoryMoveInput) (json.RawMessage, error)
	UpsertCategoryTranslation(context.Context, uint, string, CategoryTranslationInput) (json.RawMessage, error)
}

type TagService interface {
	Tags(context.Context, string, int, int) (json.RawMessage, error)
	CreateTag(context.Context, TagInput) (json.RawMessage, error)
	UpsertTagTranslation(context.Context, uint, string, TagTranslationInput) (json.RawMessage, error)
}

type APIError struct {
	Code    string
	Message string
}

func (e *APIError) Error() string { return e.Code + ": " + e.Message }

type writeOperationKey struct{}

// WithWriteOperation binds an MCP write intent to its duplicate-write guard key.
func WithWriteOperation(ctx context.Context, operationID string) context.Context {
	return context.WithValue(ctx, writeOperationKey{}, strings.TrimSpace(operationID))
}

func WriteOperationID(ctx context.Context) string {
	operationID, _ := ctx.Value(writeOperationKey{}).(string)
	return operationID
}
