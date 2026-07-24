// Package cmsclient provides the common HTTP transport for this repository's
// CMS API. Endpoint-specific clients remain responsible for their own routes,
// DTOs, and authorization policy.
package cmsclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const maxResponseBodyBytes int64 = 16 << 20

// Authorizer adds caller-specific authentication to an outgoing request.
// A nil Authorizer is appropriate for public CMS endpoints.
type Authorizer interface {
	Authorize(context.Context, *http.Request) error
}

// AuthorizerFunc adapts a function into an Authorizer.
type AuthorizerFunc func(context.Context, *http.Request) error

func (f AuthorizerFunc) Authorize(ctx context.Context, req *http.Request) error {
	return f(ctx, req)
}

// TokenProvider supplies an access token for an authenticated CMS caller.
type TokenProvider interface {
	Token(context.Context) (string, error)
}

// BearerAuthorizer adapts a token provider for CMS management endpoints.
func BearerAuthorizer(tokens TokenProvider) Authorizer {
	if tokens == nil {
		return nil
	}
	return AuthorizerFunc(func(ctx context.Context, req *http.Request) error {
		token, err := tokens.Token(ctx)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	})
}

// RequestOptions contains transport concerns shared by CMS callers.
type RequestOptions struct {
	Authorizer Authorizer
	Headers    http.Header
}

// Client performs HTTP requests against one CMS API origin.
type Client struct {
	baseURL *url.URL
	http    *http.Client
}

// New creates a client for an absolute HTTP(S) API URL. Security policies,
// such as requiring HTTPS for a particular caller, remain at the caller layer.
func New(baseURL string, httpClient *http.Client) (*Client, error) {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || u.Scheme == "" || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("CMS base URL must be an absolute HTTP URL")
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("CMS base URL must not contain a query or fragment")
	}
	if httpClient == nil {
		return nil, fmt.Errorf("CMS HTTP client is required")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath = strings.TrimRight(u.RawPath, "/")
	return &Client{baseURL: u, http: httpClient}, nil
}

// Get makes a request without a body and returns a bounded response body.
func (c *Client) Get(ctx context.Context, path string, query url.Values, options RequestOptions) ([]byte, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path, query, nil, options)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

// SendJSON sends a JSON request body. A nil payload is encoded as JSON null,
// which is required by several CMS action endpoints.
func (c *Client) SendJSON(ctx context.Context, method, path string, query url.Values, payload any, options RequestOptions) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode CMS request: %w", err)
	}
	req, err := c.newRequest(ctx, method, path, query, bytes.NewReader(body), options)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

func (c *Client) newRequest(ctx context.Context, method, path string, query url.Values, body io.Reader, options RequestOptions) (*http.Request, error) {
	u, err := c.resolve(path, query)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("create CMS request: %w", err)
	}
	for key, values := range options.Headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	if options.Authorizer != nil {
		if err := options.Authorizer.Authorize(ctx, req); err != nil {
			return nil, fmt.Errorf("authorize CMS request: %w", err)
		}
	}
	return req, nil
}

func (c *Client) resolve(path string, query url.Values) (*url.URL, error) {
	route, err := url.Parse(path)
	if err != nil || route.IsAbs() || route.Host != "" || route.RawQuery != "" || route.Fragment != "" || !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("CMS request path must be an absolute path without query or fragment")
	}
	u := *c.baseURL
	u.Path = strings.TrimRight(c.baseURL.Path, "/") + route.Path
	u.RawPath = strings.TrimRight(c.baseURL.EscapedPath(), "/") + route.EscapedPath()
	u.RawQuery = query.Encode()
	return &u, nil
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call CMS API: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read CMS response: %w", err)
	}
	if int64(len(body)) > maxResponseBodyBytes {
		return nil, fmt.Errorf("CMS response exceeds %d byte limit", maxResponseBodyBytes)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("CMS API HTTP %d", resp.StatusCode)
	}
	return body, nil
}
