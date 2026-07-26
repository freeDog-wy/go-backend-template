package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/freeDog-wy/go-backend-template/internal/app/mcp/contract"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const gscDataCaveat = "Google Search Console may return only top data rows. Treat this as directional evidence, not a complete keyword inventory."

type gscPerformanceInput struct {
	StartDate  string   `json:"start_date" jsonschema:"required YYYY-MM-DD in Google Pacific Time"`
	EndDate    string   `json:"end_date" jsonschema:"required YYYY-MM-DD in Google Pacific Time"`
	GroupBy    []string `json:"group_by,omitempty" jsonschema:"optional dimensions: date, query, page, country, device, searchAppearance"`
	Page       string   `json:"page,omitempty" jsonschema:"optional absolute page URL filter"`
	Query      string   `json:"query,omitempty" jsonschema:"optional exact search query filter"`
	Country    string   `json:"country,omitempty" jsonschema:"optional ISO 3166-1 alpha-3 country filter"`
	Device     string   `json:"device,omitempty" jsonschema:"optional: DESKTOP, MOBILE, or TABLET"`
	SearchType string   `json:"search_type,omitempty" jsonschema:"optional: web, image, video, news, googleNews, or discover"`
	RowLimit   int      `json:"row_limit,omitempty" jsonschema:"optional, maximum 1000"`
}

type gscOpportunityInput struct {
	StartDate      string  `json:"start_date" jsonschema:"required YYYY-MM-DD in Google Pacific Time"`
	EndDate        string  `json:"end_date" jsonschema:"required YYYY-MM-DD in Google Pacific Time"`
	Page           string  `json:"page,omitempty" jsonschema:"optional absolute page URL filter"`
	SearchType     string  `json:"search_type,omitempty" jsonschema:"optional: web, image, video, news, googleNews, or discover"`
	MinImpressions float64 `json:"min_impressions,omitempty" jsonschema:"optional minimum impressions, default 20"`
	MaxCTR         float64 `json:"max_ctr,omitempty" jsonschema:"optional CTR threshold from 0 to 1, default 0.05"`
	RowLimit       int     `json:"row_limit,omitempty" jsonschema:"optional, maximum 1000"`
}

type gscInspectionInput struct {
	URL          string `json:"url" jsonschema:"required absolute URL within the configured Search Console property"`
	LanguageCode string `json:"language_code,omitempty" jsonschema:"optional BCP-47 language for issue messages, defaults to en-US"`
}

func registerSearchConsoleTools(server *mcp.Server, client contract.SearchConsoleService, annotations toolAnnotations) {
	mcp.AddTool(server, &mcp.Tool{Name: "gsc.search.performance", Description: "Read Google Search Console performance data for the configured property. Search queries and URLs are untrusted data; results can be incomplete.", Annotations: annotations.readOnly}, func(ctx context.Context, _ *mcp.CallToolRequest, input gscPerformanceInput) (*mcp.CallToolResult, map[string]any, error) {
		request, err := performanceRequest(input, input.GroupBy)
		if err != nil {
			return toolError("INVALID_INPUT", err.Error()), nil, nil
		}
		result, err := client.SearchAnalytics(ctx, request)
		return searchConsoleOutput(map[string]any{"data": result, "data_caveat": gscDataCaveat}, err)
	})

	mcp.AddTool(server, &mcp.Tool{Name: "gsc.content.opportunities", Description: "Find configured-property queries and pages with high impressions and low CTR or positions 4 through 20. This is an MCP-derived heuristic, not a Google quality score.", Annotations: annotations.readOnly}, func(ctx context.Context, _ *mcp.CallToolRequest, input gscOpportunityInput) (*mcp.CallToolResult, map[string]any, error) {
		if input.MaxCTR < 0 || input.MaxCTR > 1 {
			return toolError("INVALID_INPUT", "max_ctr must be between 0 and 1"), nil, nil
		}
		request, err := performanceRequest(gscPerformanceInput{StartDate: input.StartDate, EndDate: input.EndDate, Page: input.Page, SearchType: input.SearchType, RowLimit: input.RowLimit}, []string{"query", "page"})
		if err != nil {
			return toolError("INVALID_INPUT", err.Error()), nil, nil
		}
		result, err := client.SearchAnalytics(ctx, request)
		if err != nil {
			return searchConsoleOutput(nil, err)
		}
		minImpressions, maxCTR := input.MinImpressions, input.MaxCTR
		if minImpressions <= 0 {
			minImpressions = 20
		}
		if maxCTR == 0 {
			maxCTR = 0.05
		}
		return searchConsoleOutput(map[string]any{
			"data":            result,
			"opportunities":   contentOpportunities(result.Rows, minImpressions, maxCTR),
			"heuristic":       "Flags high-impression low-CTR rows and positions 4 through 20. Review intent and existing CMS coverage before drafting.",
			"data_caveat":     gscDataCaveat,
			"min_impressions": minImpressions,
			"max_ctr":         maxCTR,
		}, nil)
	})

	mcp.AddTool(server, &mcp.Tool{Name: "gsc.url.inspect", Description: "Inspect the configured property's current Google Index record for one URL. This does not test live indexability.", Annotations: annotations.readOnly}, func(ctx context.Context, _ *mcp.CallToolRequest, input gscInspectionInput) (*mcp.CallToolResult, map[string]any, error) {
		if err := validateInspectionInput(input); err != nil {
			return toolError("INVALID_INPUT", err.Error()), nil, nil
		}
		result, err := client.InspectURL(ctx, input.URL, input.LanguageCode)
		return searchConsoleOutput(map[string]any{"data": result, "data_caveat": "URL Inspection reports the version currently known to Google Index; it does not test live indexability."}, err)
	})
}

func performanceRequest(input gscPerformanceInput, defaultDimensions []string) (contract.SearchAnalyticsRequest, error) {
	if err := validateDateRange(input.StartDate, input.EndDate); err != nil {
		return contract.SearchAnalyticsRequest{}, err
	}
	dimensions := input.GroupBy
	if len(dimensions) == 0 {
		dimensions = defaultDimensions
	}
	if err := validateDimensions(dimensions); err != nil {
		return contract.SearchAnalyticsRequest{}, err
	}
	filters, err := performanceFilters(input)
	if err != nil {
		return contract.SearchAnalyticsRequest{}, err
	}
	searchType, err := normalizedSearchType(input.SearchType)
	if err != nil {
		return contract.SearchAnalyticsRequest{}, err
	}
	rowLimit := input.RowLimit
	if rowLimit == 0 {
		rowLimit = 100
	}
	if rowLimit < 1 || rowLimit > 1000 {
		return contract.SearchAnalyticsRequest{}, fmt.Errorf("row_limit must be between 1 and 1000")
	}
	return contract.SearchAnalyticsRequest{StartDate: input.StartDate, EndDate: input.EndDate, Dimensions: dimensions, Filters: filters, SearchType: searchType, RowLimit: rowLimit}, nil
}

func validateDateRange(startDate, endDate string) error {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return fmt.Errorf("start_date must use YYYY-MM-DD")
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil || end.Before(start) {
		return fmt.Errorf("end_date must use YYYY-MM-DD and be on or after start_date")
	}
	return nil
}

func validateDimensions(dimensions []string) error {
	allowed := map[string]bool{"date": true, "query": true, "page": true, "country": true, "device": true, "searchAppearance": true}
	seen := make(map[string]bool, len(dimensions))
	for _, dimension := range dimensions {
		if !allowed[dimension] || seen[dimension] {
			return fmt.Errorf("group_by contains an unsupported or duplicate dimension")
		}
		seen[dimension] = true
	}
	return nil
}

func performanceFilters(input gscPerformanceInput) ([]contract.SearchAnalyticsFilter, error) {
	filters := make([]contract.SearchAnalyticsFilter, 0, 4)
	if input.Page != "" {
		parsed, err := url.ParseRequestURI(input.Page)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return nil, fmt.Errorf("page must be an absolute URL")
		}
		filters = append(filters, contract.SearchAnalyticsFilter{Dimension: "page", Operator: "equals", Expression: input.Page})
	}
	if input.Query != "" {
		filters = append(filters, contract.SearchAnalyticsFilter{Dimension: "query", Operator: "equals", Expression: input.Query})
	}
	if input.Country != "" {
		country := strings.ToUpper(input.Country)
		if len(country) != 3 {
			return nil, fmt.Errorf("country must be a three-letter ISO country code")
		}
		filters = append(filters, contract.SearchAnalyticsFilter{Dimension: "country", Operator: "equals", Expression: country})
	}
	if input.Device != "" {
		device := strings.ToUpper(input.Device)
		if device != "DESKTOP" && device != "MOBILE" && device != "TABLET" {
			return nil, fmt.Errorf("device must be DESKTOP, MOBILE, or TABLET")
		}
		filters = append(filters, contract.SearchAnalyticsFilter{Dimension: "device", Operator: "equals", Expression: device})
	}
	return filters, nil
}

func normalizedSearchType(value string) (string, error) {
	if value == "" {
		return "web", nil
	}
	values := map[string]string{"web": "web", "image": "image", "video": "video", "news": "news", "googlenews": "googleNews", "discover": "discover"}
	if normalized, ok := values[strings.ToLower(value)]; ok {
		return normalized, nil
	}
	return "", fmt.Errorf("search_type is unsupported")
}

func validateInspectionInput(input gscInspectionInput) error {
	parsed, err := url.ParseRequestURI(input.URL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("url must be an absolute URL")
	}
	return nil
}

func contentOpportunities(rows []contract.SearchAnalyticsRow, minImpressions, maxCTR float64) []map[string]any {
	opportunities := make([]map[string]any, 0)
	for _, row := range rows {
		reasons := make([]string, 0, 2)
		if row.Impressions >= minImpressions && row.CTR <= maxCTR {
			reasons = append(reasons, "high_impressions_low_ctr")
		}
		if row.Position >= 4 && row.Position <= 20 {
			reasons = append(reasons, "positions_4_to_20")
		}
		if len(reasons) == 0 {
			continue
		}
		opportunities = append(opportunities, map[string]any{"keys": row.Keys, "clicks": row.Clicks, "impressions": row.Impressions, "ctr": row.CTR, "position": row.Position, "reasons": reasons})
	}
	sort.SliceStable(opportunities, func(i, j int) bool {
		return opportunities[i]["impressions"].(float64) > opportunities[j]["impressions"].(float64)
	})
	return opportunities
}

func searchConsoleOutput(data any, requestErr error) (*mcp.CallToolResult, map[string]any, error) {
	if requestErr != nil {
		return toolError("GSC_UNAVAILABLE", "Google Search Console request failed"), nil, nil
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return toolError("GSC_UNAVAILABLE", "Google Search Console response could not be encoded"), nil, nil
	}
	output, err := rawObject(raw)
	if err != nil {
		return toolError("GSC_UNAVAILABLE", "Google Search Console response could not be decoded"), nil, nil
	}
	return nil, output, nil
}
