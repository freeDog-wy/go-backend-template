package cmsclient

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicClientUsesPublicContentRoutes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/public/locales":
			_, _ = w.Write([]byte(`{"success":true,"data":[{"code":"en-US","name":"English","is_enabled":true}]}`))
		case "/api/v1/public/en-US/articles/hello":
			_, _ = w.Write([]byte(`{"success":true,"data":{"locale":"en-US","slug":"hello","content_format":"markdown"}}`))
		default:
			t.Fatalf("unexpected public route %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewPublic(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	locales, err := client.ListLocales(context.Background())
	if err != nil || len(locales) != 1 || locales[0].Code != "en-US" {
		t.Fatalf("ListLocales() = %#v, %v", locales, err)
	}
	article, err := client.GetArticle(context.Background(), "en-US", "hello")
	if err != nil || article.Slug != "hello" {
		t.Fatalf("GetArticle() = %#v, %v", article, err)
	}
}

func TestAdminClientOwnsAuthenticatedWriteRoutes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/admin/cms/articles" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer access-token" || r.Header.Get("X-Correlation-ID") != "write-42" || r.Header.Get("Idempotency-Key") != "write-42" {
			t.Fatalf("headers = %#v", r.Header)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || !strings.Contains(string(body), `"title":"Draft"`) {
			t.Fatalf("body = %q, err = %v", body, err)
		}
		_, _ = w.Write([]byte(`{"success":true,"data":{"id":7}}`))
	}))
	defer server.Close()

	client, err := NewAdmin(server.URL, server.Client(), BearerAuthorizer(tokenProviderStub{}), true)
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithWriteOperation(context.Background(), "write-42")
	data, err := client.CreateArticleDraft(ctx, ArticleInput{Locale: "en-US", Title: "Draft", Slug: "draft"})
	if err != nil || string(data) != `{"id":7}` {
		t.Fatalf("CreateArticleDraft() = %s, %v", data, err)
	}
}

type tokenProviderStub struct{}

func (tokenProviderStub) Token(context.Context) (string, error) { return "access-token", nil }
