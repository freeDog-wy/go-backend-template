package cmsclient

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type writeOperationKey struct{}

// WithWriteOperation binds an idempotency key to one admin write intent.
func WithWriteOperation(ctx context.Context, operationID string) context.Context {
	return context.WithValue(ctx, writeOperationKey{}, strings.TrimSpace(operationID))
}

// WriteOperationID returns the idempotency key bound to ctx.
func WriteOperationID(ctx context.Context) string {
	operationID, _ := ctx.Value(writeOperationKey{}).(string)
	return operationID
}

// AdminClient exposes the authenticated CMS administration API.
type AdminClient struct {
	api        *Client
	authorizer Authorizer
}

func NewAdmin(baseURL string, httpClient *http.Client, authorizer Authorizer, allowInsecureHTTP bool) (*AdminClient, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	u, err := url.Parse(baseURL)
	validScheme := u != nil && (u.Scheme == "https" || (allowInsecureHTTP && u.Scheme == "http"))
	if err != nil || !validScheme || u.Host == "" {
		return nil, fmt.Errorf("CMS base URL must be an HTTPS URL")
	}
	if authorizer == nil {
		return nil, fmt.Errorf("CMS admin authorizer is required")
	}
	api, err := New(baseURL, httpClient)
	if err != nil {
		return nil, err
	}
	return &AdminClient{api: api, authorizer: authorizer}, nil
}

func (c *AdminClient) Health(ctx context.Context) (json.RawMessage, error) {
	live, err := c.getPublic(ctx, "/healthz")
	if err != nil {
		return nil, err
	}
	ready, err := c.getPublic(ctx, "/readyz")
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]json.RawMessage{"live": live, "ready": ready})
}

func (c *AdminClient) Locales(ctx context.Context) (json.RawMessage, error) {
	return c.getAdmin(ctx, "/api/v1/admin/cms/locales", nil)
}

func (c *AdminClient) CreateLocale(ctx context.Context, input LocaleCreateInput) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPost, "/api/v1/admin/cms/locales", input)
}

func (c *AdminClient) UpdateLocale(ctx context.Context, code string, input LocaleUpdateInput) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPatch, "/api/v1/admin/cms/locales/"+url.PathEscape(code), input)
}

func (c *AdminClient) Articles(ctx context.Context, locale, status string, page, perPage int) (json.RawMessage, error) {
	query := pageQuery(locale, page, perPage)
	if status = strings.TrimSpace(status); status != "" {
		query.Set("status", status)
	}
	return c.getAdmin(ctx, "/api/v1/admin/cms/articles", query)
}

func (c *AdminClient) ArticleTranslation(ctx context.Context, articleID uint, locale string) (json.RawMessage, error) {
	if articleID == 0 || strings.TrimSpace(locale) == "" {
		return nil, &APIError{Code: "INVALID_INPUT", Message: "article_id and locale are required"}
	}
	return c.getAdmin(ctx, "/api/v1/admin/cms/articles/"+uintPath(articleID)+"/translations/"+url.PathEscape(locale), nil)
}

func (c *AdminClient) CreateArticleDraft(ctx context.Context, input ArticleInput) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPost, "/api/v1/admin/cms/articles", input)
}

func (c *AdminClient) CreateArticleTranslation(ctx context.Context, articleID uint, input ArticleInput) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPost, "/api/v1/admin/cms/articles/"+uintPath(articleID)+"/translations", input)
}

func (c *AdminClient) UpdateArticleTranslation(ctx context.Context, articleID uint, locale string, input ArticleInput) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPut, "/api/v1/admin/cms/articles/"+uintPath(articleID)+"/translations/"+url.PathEscape(locale), input)
}

func (c *AdminClient) ReplaceArticleCategories(ctx context.Context, articleID uint, categoryIDs []uint, primaryCategoryID *uint) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPut, "/api/v1/admin/cms/articles/"+uintPath(articleID)+"/categories", map[string]any{"category_ids": categoryIDs, "primary_category_id": primaryCategoryID})
}

func (c *AdminClient) ReplaceArticleTags(ctx context.Context, articleID uint, tagIDs []uint) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPut, "/api/v1/admin/cms/articles/"+uintPath(articleID)+"/tags", map[string]any{"tag_ids": tagIDs})
}

func (c *AdminClient) PreviewPublish(ctx context.Context, articleID uint, locale string) (json.RawMessage, error) {
	return c.getAdmin(ctx, "/api/v1/admin/cms/articles/"+uintPath(articleID)+"/translations/"+url.PathEscape(locale)+"/publish-preview", nil)
}

func (c *AdminClient) PublishArticleTranslation(ctx context.Context, articleID uint, locale string) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPost, "/api/v1/admin/cms/articles/"+uintPath(articleID)+"/translations/"+url.PathEscape(locale)+"/publish", nil)
}

func (c *AdminClient) ArchiveArticleTranslation(ctx context.Context, articleID uint, locale string) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPost, "/api/v1/admin/cms/articles/"+uintPath(articleID)+"/translations/"+url.PathEscape(locale)+"/archive", nil)
}

func (c *AdminClient) RestoreArticle(ctx context.Context, articleID uint) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPost, "/api/v1/admin/cms/articles/"+uintPath(articleID)+"/restore", nil)
}

func (c *AdminClient) SetArticleCover(ctx context.Context, articleID uint, mediaID *uint) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPut, "/api/v1/admin/cms/articles/"+uintPath(articleID)+"/cover", map[string]any{"media_id": mediaID})
}

func (c *AdminClient) Categories(ctx context.Context, locale string) (json.RawMessage, error) {
	data, err := c.getAdmin(ctx, "/api/v1/admin/cms/categories", url.Values{"locale": {locale}})
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]json.RawMessage{"data": data})
}

func (c *AdminClient) CreateCategory(ctx context.Context, input CategoryInput) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPost, "/api/v1/admin/cms/categories", input)
}

func (c *AdminClient) UpdateCategory(ctx context.Context, categoryID uint, input CategoryStateInput) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPatch, "/api/v1/admin/cms/categories/"+uintPath(categoryID), input)
}

func (c *AdminClient) MoveCategory(ctx context.Context, categoryID uint, input CategoryMoveInput) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPatch, "/api/v1/admin/cms/categories/"+uintPath(categoryID)+"/move", input)
}

func (c *AdminClient) UpsertCategoryTranslation(ctx context.Context, categoryID uint, locale string, input CategoryTranslationInput) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPut, "/api/v1/admin/cms/categories/"+uintPath(categoryID)+"/translations/"+url.PathEscape(locale), input)
}

func (c *AdminClient) Tags(ctx context.Context, locale string, page, perPage int) (json.RawMessage, error) {
	return c.getAdmin(ctx, "/api/v1/admin/cms/tags", pageQuery(locale, page, perPage))
}

func (c *AdminClient) CreateTag(ctx context.Context, input TagInput) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPost, "/api/v1/admin/cms/tags", input)
}

func (c *AdminClient) UpsertTagTranslation(ctx context.Context, tagID uint, locale string, input TagTranslationInput) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPut, "/api/v1/admin/cms/tags/"+uintPath(tagID)+"/translations/"+url.PathEscape(locale), input)
}

func (c *AdminClient) getPublic(ctx context.Context, path string) (json.RawMessage, error) {
	body, err := c.api.Get(ctx, path, nil, RequestOptions{})
	if err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}

func (c *AdminClient) getAdmin(ctx context.Context, path string, query url.Values) (json.RawMessage, error) {
	body, err := c.api.Get(ctx, path, query, RequestOptions{Authorizer: c.authorizer})
	if err != nil {
		return nil, err
	}
	return unwrapAdminEnvelope(body)
}

func (c *AdminClient) write(ctx context.Context, method, path string, payload any) (json.RawMessage, error) {
	operationID := WriteOperationID(ctx)
	if operationID == "" {
		operationID = correlationID()
	}
	headers := http.Header{}
	headers.Set("X-Correlation-ID", operationID)
	headers.Set("Idempotency-Key", operationID)
	body, err := c.api.SendJSON(ctx, method, path, nil, payload, RequestOptions{Authorizer: c.authorizer, Headers: headers})
	if err != nil {
		return nil, err
	}
	return unwrapAdminEnvelope(body)
}

func unwrapAdminEnvelope(body []byte) (json.RawMessage, error) {
	envelope, err := DecodeEnvelope(body)
	if err != nil {
		return nil, err
	}
	if len(envelope.Meta) == 0 || string(envelope.Meta) == "null" {
		return envelope.Data, nil
	}
	return json.Marshal(map[string]json.RawMessage{"data": envelope.Data, "meta": envelope.Meta})
}

func pageQuery(locale string, page, perPage int) url.Values {
	values := url.Values{"locale": {locale}}
	if page > 0 {
		values.Set("page", strconv.Itoa(page))
	}
	if perPage > 0 {
		values.Set("per_page", strconv.Itoa(perPage))
	}
	return values
}

func uintPath(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}

func correlationID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "cmsclient"
	}
	return hex.EncodeToString(buf)
}
