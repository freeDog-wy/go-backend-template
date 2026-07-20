package client

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/freeDog-wy/go-backend-template/internal/app/mcp/contract"
)

type ArticleInput = contract.ArticleInput
type CategoryInput = contract.CategoryInput
type CategoryStateInput = contract.CategoryStateInput
type CategoryMoveInput = contract.CategoryMoveInput
type CategoryTranslationInput = contract.CategoryTranslationInput
type TagInput = contract.TagInput
type TagTranslationInput = contract.TagTranslationInput
type LocaleCreateInput = contract.LocaleCreateInput
type LocaleUpdateInput = contract.LocaleUpdateInput

type TokenProvider interface {
	Token(context.Context) (string, error)
}

// WithWriteOperation binds one MCP write intent to a stable idempotency key.
// Callers must reuse the same value only when retrying an uncertain outcome.
func WithWriteOperation(ctx context.Context, operationID string) context.Context {
	return contract.WithWriteOperation(ctx, strings.TrimSpace(operationID))
}

type Client struct {
	baseURL string
	http    *http.Client
	tokens  TokenProvider
}

var (
	_ contract.SiteReader      = (*Client)(nil)
	_ contract.LocaleService   = (*Client)(nil)
	_ contract.ArticleService  = (*Client)(nil)
	_ contract.CategoryService = (*Client)(nil)
	_ contract.TagService      = (*Client)(nil)
)

type APIError = contract.APIError

func New(baseURL string, httpClient *http.Client, tokens TokenProvider, allowInsecureHTTP bool) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	u, err := url.Parse(baseURL)
	validScheme := u != nil && (u.Scheme == "https" || (allowInsecureHTTP && u.Scheme == "http"))
	if err != nil || !validScheme || u.Host == "" {
		return nil, fmt.Errorf("cms base URL must be an HTTPS URL")
	}
	if httpClient == nil || tokens == nil {
		return nil, fmt.Errorf("mcp HTTP client and token provider are required")
	}
	return &Client{baseURL: baseURL, http: httpClient, tokens: tokens}, nil
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

func (c *Client) getPublic(ctx context.Context, path string) (json.RawMessage, error) {
	return c.request(ctx, path, nil, false)
}

func (c *Client) getAdmin(ctx context.Context, path string, query url.Values) (json.RawMessage, error) {
	return c.request(ctx, path, query, true)
}

func (c *Client) request(ctx context.Context, path string, query url.Values, auth bool) (json.RawMessage, error) {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, err
	}
	u.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	return c.do(req, auth)
}

func (c *Client) write(ctx context.Context, method, path string, payload any) (json.RawMessage, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode CMS request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	operationID := contract.WriteOperationID(ctx)
	if operationID == "" {
		operationID = correlationID()
	}
	req.Header.Set("X-Correlation-ID", operationID)
	req.Header.Set("Idempotency-Key", operationID)
	return c.do(req, true)
}

func (c *Client) do(req *http.Request, auth bool) (json.RawMessage, error) {
	if auth {
		token, err := c.tokens.Token(req.Context())
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call CMS API: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("CMS API HTTP %d", resp.StatusCode)
	}
	if !auth {
		return json.RawMessage(body), nil
	}
	var envelope struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
		Meta    json.RawMessage `json:"meta"`
		Error   *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("decode CMS response: %w", err)
	}
	if !envelope.Success {
		if envelope.Error == nil {
			return nil, &APIError{Code: "UNKNOWN", Message: "CMS request failed"}
		}
		return nil, &APIError{Code: envelope.Error.Code, Message: envelope.Error.Message}
	}
	if len(envelope.Meta) == 0 || string(envelope.Meta) == "null" {
		return envelope.Data, nil
	}
	return json.Marshal(map[string]json.RawMessage{"data": envelope.Data, "meta": envelope.Meta})
}

func correlationID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "mcp"
	}
	return hex.EncodeToString(buf)
}
