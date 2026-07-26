// Package gsc provides a narrow, read-only Google Search Console adapter.
package gsc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/freeDog-wy/go-backend-template/internal/app/mcp/contract"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/jwt"
)

const (
	readonlyScope     = "https://www.googleapis.com/auth/webmasters.readonly"
	webmastersBaseURL = "https://www.googleapis.com/webmasters/v3"
	inspectionURL     = "https://searchconsole.googleapis.com/v1/urlInspection/index:inspect"
)

type Client struct {
	property          string
	httpClient        *http.Client
	webmastersBaseURL string
	inspectionURL     string
}

type serviceAccountCredentials struct {
	Type        string `json:"type"`
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	TokenURI    string `json:"token_uri"`
}

func New(property, credentialsFile string, httpClient *http.Client) (*Client, error) {
	credentials, err := os.ReadFile(strings.TrimSpace(credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("read GSC service account credentials: %w", err)
	}
	var serviceAccount serviceAccountCredentials
	if err := json.Unmarshal(credentials, &serviceAccount); err != nil {
		return nil, fmt.Errorf("decode GSC service account credentials: %w", err)
	}
	if serviceAccount.Type != "service_account" || strings.TrimSpace(serviceAccount.ClientEmail) == "" || strings.TrimSpace(serviceAccount.PrivateKey) == "" || strings.TrimSpace(serviceAccount.TokenURI) == "" {
		return nil, fmt.Errorf("GSC service account credentials are incomplete")
	}

	tokenSource := (&jwt.Config{
		Email:      serviceAccount.ClientEmail,
		PrivateKey: []byte(serviceAccount.PrivateKey),
		TokenURL:   serviceAccount.TokenURI,
		Scopes:     []string{readonlyScope},
	}).TokenSource(context.Background())
	return NewWithTokenSource(property, tokenSource, httpClient)
}

func NewWithTokenSource(property string, tokenSource oauth2.TokenSource, httpClient *http.Client) (*Client, error) {
	property = strings.TrimSpace(property)
	if property == "" || tokenSource == nil || httpClient == nil {
		return nil, fmt.Errorf("GSC property, token source, and HTTP client are required")
	}
	client := *httpClient
	client.Transport = &oauth2.Transport{Source: tokenSource, Base: httpClient.Transport}
	return &Client{
		property:          property,
		httpClient:        &client,
		webmastersBaseURL: webmastersBaseURL,
		inspectionURL:     inspectionURL,
	}, nil
}

func (c *Client) SearchAnalytics(ctx context.Context, input contract.SearchAnalyticsRequest) (*contract.SearchAnalyticsResult, error) {
	request := struct {
		StartDate             string                       `json:"startDate"`
		EndDate               string                       `json:"endDate"`
		Dimensions            []string                     `json:"dimensions,omitempty"`
		Type                  string                       `json:"type,omitempty"`
		DimensionFilterGroups []searchAnalyticsFilterGroup `json:"dimensionFilterGroups,omitempty"`
		RowLimit              int                          `json:"rowLimit,omitempty"`
	}{
		StartDate:  input.StartDate,
		EndDate:    input.EndDate,
		Dimensions: input.Dimensions,
		Type:       input.SearchType,
		RowLimit:   input.RowLimit,
	}
	if len(input.Filters) > 0 {
		filters := make([]searchAnalyticsFilter, 0, len(input.Filters))
		for _, filter := range input.Filters {
			filters = append(filters, searchAnalyticsFilter{Dimension: filter.Dimension, Operator: filter.Operator, Expression: filter.Expression})
		}
		request.DimensionFilterGroups = []searchAnalyticsFilterGroup{{GroupType: "and", Filters: filters}}
	}

	var response struct {
		Rows []contract.SearchAnalyticsRow `json:"rows"`
	}
	endpoint := c.webmastersBaseURL + "/sites/" + url.PathEscape(c.property) + "/searchAnalytics/query"
	if err := c.doJSON(ctx, http.MethodPost, endpoint, request, &response); err != nil {
		return nil, err
	}
	return &contract.SearchAnalyticsResult{Property: c.property, Rows: response.Rows}, nil
}

func (c *Client) InspectURL(ctx context.Context, inspectionURL, languageCode string) (*contract.URLInspectionResult, error) {
	request := struct {
		InspectionURL string `json:"inspectionUrl"`
		SiteURL       string `json:"siteUrl"`
		LanguageCode  string `json:"languageCode,omitempty"`
	}{InspectionURL: inspectionURL, SiteURL: c.property, LanguageCode: languageCode}
	var response struct {
		InspectionResult map[string]any `json:"inspectionResult"`
	}
	if err := c.doJSON(ctx, http.MethodPost, c.inspectionURL, request, &response); err != nil {
		return nil, err
	}
	return &contract.URLInspectionResult{InspectionURL: inspectionURL, Property: c.property, InspectionResult: response.InspectionResult}, nil
}

type searchAnalyticsFilterGroup struct {
	GroupType string                  `json:"groupType"`
	Filters   []searchAnalyticsFilter `json:"filters"`
}

type searchAnalyticsFilter struct {
	Dimension  string `json:"dimension"`
	Operator   string `json:"operator"`
	Expression string `json:"expression"`
}

func (c *Client) doJSON(ctx context.Context, method, endpoint string, input, output any) error {
	body, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("encode GSC request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create GSC request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request GSC: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("GSC API HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(message)))
	}
	if err := json.NewDecoder(resp.Body).Decode(output); err != nil {
		return fmt.Errorf("decode GSC response: %w", err)
	}
	return nil
}

var _ contract.SearchConsoleService = (*Client)(nil)
