package sitegen

import (
	"context"
	"fmt"
	"net/http"

	"github.com/freeDog-wy/go-backend-template/internal/app/pkg/cmsclient"
)

// Client adapts the CMS public API to the static build's full-pagination
// snapshot requirements. CMS route definitions live in cmsclient.
type Client struct {
	public  *cmsclient.PublicClient
	perPage int
}

func NewClient(cfg Config) *Client {
	public, err := cmsclient.NewPublic(cfg.APIBaseURL.String(), &http.Client{Timeout: cfg.HTTPTimeout})
	if err != nil {
		panic(fmt.Sprintf("create sitegen CMS client: %v", err))
	}
	return &Client{public: public, perPage: cfg.PerPage}
}

func (c *Client) ListLocales(ctx context.Context) ([]Locale, error) {
	return c.public.ListLocales(ctx)
}

func (c *Client) ListCategories(ctx context.Context, locale string) ([]Category, error) {
	return c.public.ListCategories(ctx, locale)
}

func (c *Client) ListArticles(ctx context.Context, locale string) ([]ArticleListItem, error) {
	return listAll(ctx, func(ctx context.Context, page int) ([]ArticleListItem, *pageMeta, error) {
		return c.public.ListArticles(ctx, locale, page, c.perPage)
	})
}

func (c *Client) GetArticle(ctx context.Context, locale, slug string) (Article, error) {
	return c.public.GetArticle(ctx, locale, slug)
}

func (c *Client) ListTags(ctx context.Context, locale string) ([]Tag, error) {
	return listAll(ctx, func(ctx context.Context, page int) ([]Tag, *pageMeta, error) {
		return c.public.ListTags(ctx, locale, page, c.perPage)
	})
}

func (c *Client) ListCategoryArticles(ctx context.Context, locale, slug string) ([]ArticleListItem, error) {
	return listAll(ctx, func(ctx context.Context, page int) ([]ArticleListItem, *pageMeta, error) {
		return c.public.ListCategoryArticles(ctx, locale, slug, page, c.perPage)
	})
}

func (c *Client) ListTagArticles(ctx context.Context, locale, slug string) ([]ArticleListItem, error) {
	return listAll(ctx, func(ctx context.Context, page int) ([]ArticleListItem, *pageMeta, error) {
		return c.public.ListTagArticles(ctx, locale, slug, page, c.perPage)
	})
}

func (c *Client) ListSitemapEntries(ctx context.Context, locale string) ([]SitemapEntry, error) {
	return listAll(ctx, func(ctx context.Context, page int) ([]SitemapEntry, *pageMeta, error) {
		return c.public.ListSitemapEntries(ctx, locale, page, c.perPage)
	})
}

func (c *Client) ListRedirects(ctx context.Context, locale string) ([]Redirect, error) {
	return listAll(ctx, func(ctx context.Context, page int) ([]Redirect, *pageMeta, error) {
		return c.public.ListRedirects(ctx, locale, page, c.perPage)
	})
}

func listAll[T any](ctx context.Context, getPage func(context.Context, int) ([]T, *pageMeta, error)) ([]T, error) {
	first, meta, err := getPage(ctx, 1)
	if err != nil {
		return nil, err
	}
	if err := validateMeta(meta, 1); err != nil {
		return nil, err
	}
	all := append([]T(nil), first...)
	for page := 2; page <= meta.TotalPages; page++ {
		items, current, err := getPage(ctx, page)
		if err != nil {
			return nil, err
		}
		if err := validateMeta(current, page); err != nil {
			return nil, err
		}
		if current.TotalPages != meta.TotalPages || current.Total != meta.Total {
			return nil, fmt.Errorf("pagination metadata changed while reading page %d", page)
		}
		all = append(all, items...)
	}
	if int64(len(all)) != meta.Total {
		return nil, fmt.Errorf("pagination returned %d items, expected %d", len(all), meta.Total)
	}
	return all, nil
}

func validateMeta(meta *pageMeta, requestedPage int) error {
	if meta == nil {
		return fmt.Errorf("pagination metadata is missing")
	}
	if meta.Page != requestedPage || meta.PerPage < 1 || meta.Total < 0 || meta.TotalPages < 0 {
		return fmt.Errorf("invalid pagination metadata for page %d", requestedPage)
	}
	expectedPages := 0
	if meta.Total > 0 {
		expectedPages = int((meta.Total + int64(meta.PerPage) - 1) / int64(meta.PerPage))
	}
	if meta.TotalPages != expectedPages {
		return fmt.Errorf("invalid pagination total_pages for page %d", requestedPage)
	}
	return nil
}
