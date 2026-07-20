package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

func (c *Client) Locales(ctx context.Context) (json.RawMessage, error) {
	return c.getAdmin(ctx, "/api/v1/admin/cms/locales", nil)
}

func (c *Client) CreateLocale(ctx context.Context, input LocaleCreateInput) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPost, "/api/v1/admin/cms/locales", input)
}

func (c *Client) UpdateLocale(ctx context.Context, code string, input LocaleUpdateInput) (json.RawMessage, error) {
	return c.write(ctx, http.MethodPatch, "/api/v1/admin/cms/locales/"+url.PathEscape(code), input)
}
