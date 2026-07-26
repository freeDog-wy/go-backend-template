package gsc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/freeDog-wy/go-backend-template/internal/app/mcp/contract"
	"golang.org/x/oauth2"
)

func TestSearchAnalyticsAndInspection(t *testing.T) {
	requests := make([]map[string]any, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		requests = append(requests, body)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/urlInspection/index:inspect" {
			_, _ = w.Write([]byte(`{"inspectionResult":{"indexStatusResult":{"coverageState":"Submitted and indexed"}}}`))
			return
		}
		_, _ = w.Write([]byte(`{"rows":[{"keys":["query","https://example.com/a"],"clicks":2,"impressions":10,"ctr":0.2,"position":7}]}`))
	}))
	defer server.Close()

	client, err := NewWithTokenSource("sc-domain:example.com", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	client.webmastersBaseURL = server.URL
	client.inspectionURL = server.URL + "/urlInspection/index:inspect"

	performance, err := client.SearchAnalytics(context.Background(), contract.SearchAnalyticsRequest{StartDate: "2026-06-01", EndDate: "2026-06-02", Dimensions: []string{"query", "page"}, Filters: []contract.SearchAnalyticsFilter{{Dimension: "page", Operator: "equals", Expression: "https://example.com/a"}}, SearchType: "web", RowLimit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(performance.Rows) != 1 || performance.Rows[0].Position != 7 || performance.Property != "sc-domain:example.com" {
		t.Fatalf("performance = %#v", performance)
	}

	inspection, err := client.InspectURL(context.Background(), "https://example.com/a", "en-US")
	if err != nil {
		t.Fatal(err)
	}
	if inspection.InspectionResult["indexStatusResult"] == nil || len(requests) != 2 {
		t.Fatalf("inspection = %#v, requests = %#v", inspection, requests)
	}
}
