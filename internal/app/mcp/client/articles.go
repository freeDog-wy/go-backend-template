package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func (c *Client) Articles(ctx context.Context, locale, status string, page, perPage int) (json.RawMessage, error) {
	query := pageQuery(locale, page, perPage)
	if status = strings.TrimSpace(status); status != "" {
		query.Set("status", status)
	}
	return c.getAdmin(ctx, "/api/v1/admin/cms/articles", query)
}

func (c *Client) ArticleTranslation(ctx context.Context, articleID uint, locale string) (json.RawMessage, error) {
	if articleID == 0 || strings.TrimSpace(locale) == "" {
		return nil, &APIError{Code: "INVALID_INPUT", Message: "article_id and locale are required"}
	}
	return c.getAdmin(ctx, "/api/v1/admin/cms/articles/"+strconv.FormatUint(uint64(articleID), 10)+"/translations/"+url.PathEscape(locale), nil)
}

func (c *Client) CreateArticleDraft(ctx context.Context, input ArticleInput) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPost, "/api/v1/admin/cms/articles", input)
}

func (c *Client) CreateArticleTranslation(ctx context.Context, articleID uint, input ArticleInput) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPost, "/api/v1/admin/cms/articles/"+strconv.FormatUint(uint64(articleID), 10)+"/translations", input)
}

func (c *Client) UpdateArticleTranslation(ctx context.Context, articleID uint, locale string, input ArticleInput) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPut, "/api/v1/admin/cms/articles/"+strconv.FormatUint(uint64(articleID), 10)+"/translations/"+url.PathEscape(locale), input)
}

func (c *Client) ReplaceArticleCategories(ctx context.Context, articleID uint, categoryIDs []uint, primaryCategoryID *uint) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPut, "/api/v1/admin/cms/articles/"+strconv.FormatUint(uint64(articleID), 10)+"/categories", map[string]any{"category_ids": categoryIDs, "primary_category_id": primaryCategoryID})
}

func (c *Client) ReplaceArticleTags(ctx context.Context, articleID uint, tagIDs []uint) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPut, "/api/v1/admin/cms/articles/"+strconv.FormatUint(uint64(articleID), 10)+"/tags", map[string]any{"tag_ids": tagIDs})
}

func (c *Client) PreviewPublish(ctx context.Context, articleID uint, locale string) (json.RawMessage, error) {
	return c.getAdmin(ctx, "/api/v1/admin/cms/articles/"+strconv.FormatUint(uint64(articleID), 10)+"/translations/"+url.PathEscape(locale)+"/publish-preview", nil)
}

func (c *Client) PublishArticleTranslation(ctx context.Context, articleID uint, locale string) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPost, "/api/v1/admin/cms/articles/"+strconv.FormatUint(uint64(articleID), 10)+"/translations/"+url.PathEscape(locale)+"/publish", nil)
}

func (c *Client) ArchiveArticleTranslation(ctx context.Context, articleID uint, locale string) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPost, "/api/v1/admin/cms/articles/"+strconv.FormatUint(uint64(articleID), 10)+"/translations/"+url.PathEscape(locale)+"/archive", nil)
}

func (c *Client) RestoreArticle(ctx context.Context, articleID uint) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPost, "/api/v1/admin/cms/articles/"+strconv.FormatUint(uint64(articleID), 10)+"/restore", nil)
}

func (c *Client) SetArticleCover(ctx context.Context, articleID uint, mediaID *uint) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPut, "/api/v1/admin/cms/articles/"+strconv.FormatUint(uint64(articleID), 10)+"/cover", map[string]any{"media_id": mediaID})
}
