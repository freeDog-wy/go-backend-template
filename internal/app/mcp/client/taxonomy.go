package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

func (c *Client) Categories(ctx context.Context, locale string) (json.RawMessage, error) {
	data, err := c.getAdmin(ctx, "/api/v1/admin/cms/categories", url.Values{"locale": {locale}})
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]json.RawMessage{"data": data})
}

func (c *Client) Tags(ctx context.Context, locale string, page, perPage int) (json.RawMessage, error) {
	return c.getAdmin(ctx, "/api/v1/admin/cms/tags", pageQuery(locale, page, perPage))
}

func (c *Client) CreateCategory(ctx context.Context, input CategoryInput) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPost, "/api/v1/admin/cms/categories", input)
}

func (c *Client) UpdateCategory(ctx context.Context, categoryID uint, input CategoryStateInput) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPatch, "/api/v1/admin/cms/categories/"+strconv.FormatUint(uint64(categoryID), 10), input)
}

func (c *Client) MoveCategory(ctx context.Context, categoryID uint, input CategoryMoveInput) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPatch, "/api/v1/admin/cms/categories/"+strconv.FormatUint(uint64(categoryID), 10)+"/move", input)
}

func (c *Client) UpsertCategoryTranslation(ctx context.Context, categoryID uint, locale string, input CategoryTranslationInput) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPut, "/api/v1/admin/cms/categories/"+strconv.FormatUint(uint64(categoryID), 10)+"/translations/"+url.PathEscape(locale), input)
}

func (c *Client) CreateTag(ctx context.Context, input TagInput) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPost, "/api/v1/admin/cms/tags", input)
}

func (c *Client) UpsertTagTranslation(ctx context.Context, tagID uint, locale string, input TagTranslationInput) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPut, "/api/v1/admin/cms/tags/"+strconv.FormatUint(uint64(tagID), 10)+"/translations/"+url.PathEscape(locale), input)
}
