package contract

import "context"

// SearchConsoleService exposes the read-only Google Search Console operations
// used by the MCP adapter.
type SearchConsoleService interface {
	SearchAnalytics(context.Context, SearchAnalyticsRequest) (*SearchAnalyticsResult, error)
	InspectURL(context.Context, string, string) (*URLInspectionResult, error)
}

type SearchAnalyticsRequest struct {
	StartDate  string
	EndDate    string
	Dimensions []string
	Filters    []SearchAnalyticsFilter
	SearchType string
	RowLimit   int
}

type SearchAnalyticsFilter struct {
	Dimension  string
	Operator   string
	Expression string
}

type SearchAnalyticsResult struct {
	Property string               `json:"property"`
	Rows     []SearchAnalyticsRow `json:"rows"`
}

type SearchAnalyticsRow struct {
	Keys        []string `json:"keys"`
	Clicks      float64  `json:"clicks"`
	Impressions float64  `json:"impressions"`
	CTR         float64  `json:"ctr"`
	Position    float64  `json:"position"`
}

type URLInspectionResult struct {
	InspectionURL    string         `json:"inspection_url"`
	Property         string         `json:"property"`
	InspectionResult map[string]any `json:"inspection_result"`
}
