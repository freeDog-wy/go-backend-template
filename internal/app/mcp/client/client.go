// Package client provides a deprecated compatibility entry point. The complete
// CMS endpoint implementation lives in app/pkg/cmsclient.
package client

import (
	"context"
	"net/http"

	"github.com/freeDog-wy/go-backend-template/internal/app/mcp/contract"
	"github.com/freeDog-wy/go-backend-template/internal/app/pkg/cmsclient"
)

type ArticleInput = cmsclient.ArticleInput
type CategoryInput = cmsclient.CategoryInput
type CategoryStateInput = cmsclient.CategoryStateInput
type CategoryMoveInput = cmsclient.CategoryMoveInput
type CategoryTranslationInput = cmsclient.CategoryTranslationInput
type TagInput = cmsclient.TagInput
type TagTranslationInput = cmsclient.TagTranslationInput
type LocaleCreateInput = cmsclient.LocaleCreateInput
type LocaleUpdateInput = cmsclient.LocaleUpdateInput
type TokenProvider = cmsclient.TokenProvider

// Client is deprecated: use cmsclient.AdminClient.
type Client = cmsclient.AdminClient
type APIError = cmsclient.APIError

// New is deprecated: use cmsclient.NewAdmin with cmsclient.BearerAuthorizer.
func New(baseURL string, httpClient *http.Client, tokens TokenProvider, allowInsecureHTTP bool) (*Client, error) {
	return cmsclient.NewAdmin(baseURL, httpClient, cmsclient.BearerAuthorizer(tokens), allowInsecureHTTP)
}

func WithWriteOperation(ctx context.Context, operationID string) context.Context {
	return contract.WithWriteOperation(ctx, operationID)
}
