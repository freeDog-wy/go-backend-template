package cmsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// PublicClient exposes only the CMS public content API.
type PublicClient struct {
	api *Client
}

func NewPublic(baseURL string, httpClient *http.Client) (*PublicClient, error) {
	api, err := New(baseURL, httpClient)
	if err != nil {
		return nil, err
	}
	return &PublicClient{api: api}, nil
}

func (c *PublicClient) ListLocales(ctx context.Context) ([]Locale, error) {
	return publicData[[]Locale](ctx, c, "/api/v1/public/locales", nil)
}

func (c *PublicClient) ListCategories(ctx context.Context, locale string) ([]Category, error) {
	return publicData[[]Category](ctx, c, publicPath(locale, "/categories"), nil)
}

func (c *PublicClient) ListArticles(ctx context.Context, locale string, page, perPage int) ([]ArticleListItem, *PageMeta, error) {
	return publicPage[ArticleListItem](ctx, c, publicPath(locale, "/articles"), page, perPage)
}

func (c *PublicClient) GetArticle(ctx context.Context, locale, slug string) (Article, error) {
	return publicData[Article](ctx, c, publicPath(locale, "/articles/"+url.PathEscape(slug)), nil)
}

func (c *PublicClient) ListTags(ctx context.Context, locale string, page, perPage int) ([]Tag, *PageMeta, error) {
	return publicPage[Tag](ctx, c, publicPath(locale, "/tags"), page, perPage)
}

func (c *PublicClient) ListCategoryArticles(ctx context.Context, locale, slug string, page, perPage int) ([]ArticleListItem, *PageMeta, error) {
	return publicPage[ArticleListItem](ctx, c, publicPath(locale, "/categories/"+url.PathEscape(slug)+"/articles"), page, perPage)
}

func (c *PublicClient) ListTagArticles(ctx context.Context, locale, slug string, page, perPage int) ([]ArticleListItem, *PageMeta, error) {
	return publicPage[ArticleListItem](ctx, c, publicPath(locale, "/tags/"+url.PathEscape(slug)+"/articles"), page, perPage)
}

func (c *PublicClient) ListSitemapEntries(ctx context.Context, locale string, page, perPage int) ([]SitemapEntry, *PageMeta, error) {
	return publicPage[SitemapEntry](ctx, c, publicPath(locale, "/sitemap-entries"), page, perPage)
}

func (c *PublicClient) ListRedirects(ctx context.Context, locale string, page, perPage int) ([]Redirect, *PageMeta, error) {
	return publicPage[Redirect](ctx, c, publicPath(locale, "/redirects"), page, perPage)
}

func publicPath(locale, suffix string) string {
	return "/api/v1/public/" + url.PathEscape(locale) + suffix
}

func publicData[T any](ctx context.Context, c *PublicClient, path string, query url.Values) (T, error) {
	data, _, err := publicResponse[T](ctx, c, path, query)
	return data, err
}

func publicPage[T any](ctx context.Context, c *PublicClient, path string, page, perPage int) ([]T, *PageMeta, error) {
	query := url.Values{}
	query.Set("page", strconv.Itoa(page))
	query.Set("per_page", strconv.Itoa(perPage))
	return publicResponse[[]T](ctx, c, path, query)
}

func publicResponse[T any](ctx context.Context, c *PublicClient, path string, query url.Values) (T, *PageMeta, error) {
	var zero T
	body, err := c.api.Get(ctx, path, query, RequestOptions{})
	if err != nil {
		return zero, nil, fmt.Errorf("request CMS API %s: %w", path, err)
	}
	envelope, err := DecodeEnvelope(body)
	if err != nil {
		return zero, nil, err
	}
	var data T
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return zero, nil, fmt.Errorf("decode CMS response data %s: %w", path, err)
	}
	var meta *PageMeta
	if len(envelope.Meta) > 0 && string(envelope.Meta) != "null" {
		if err := json.Unmarshal(envelope.Meta, &meta); err != nil {
			return zero, nil, fmt.Errorf("decode CMS response metadata %s: %w", path, err)
		}
	}
	return data, meta, nil
}
