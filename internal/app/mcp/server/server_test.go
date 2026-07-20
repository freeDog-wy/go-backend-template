package server

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/freeDog-wy/go-backend-template/internal/app/mcp/contract"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestLocaleCodes(t *testing.T) {
	codes, err := localeCodes(json.RawMessage(`[{"code":"zh-CN"},{"code":""},{"code":"en-US"}]`))
	if err != nil {
		t.Fatal(err)
	}
	if len(codes) != 2 || codes[0] != "zh-CN" || codes[1] != "en-US" {
		t.Fatalf("localeCodes() = %v", codes)
	}
}

func TestResourceURIParsing(t *testing.T) {
	categoryReq := &mcp.ReadResourceRequest{Params: &mcp.ReadResourceParams{URI: "cms://taxonomy/categories/zh-CN"}}
	if uri, locale, err := categoryResourceURI(categoryReq); err != nil || uri != categoryReq.Params.URI || locale != "zh-CN" {
		t.Fatalf("category resource = (%q, %q, %v)", uri, locale, err)
	}
	if _, _, err := categoryResourceURI(&mcp.ReadResourceRequest{Params: &mcp.ReadResourceParams{URI: "cms://taxonomy/categories/zh-CN?page=2"}}); err == nil {
		t.Fatal("category resource accepted query parameters")
	}

	articleReq := &mcp.ReadResourceRequest{Params: &mcp.ReadResourceParams{URI: "cms://articles/7/translations/zh-CN"}}
	if uri, articleID, locale, err := articleTranslationResourceURI(articleReq); err != nil || uri != articleReq.Params.URI || articleID != 7 || locale != "zh-CN" {
		t.Fatalf("article resource = (%q, %d, %q, %v)", uri, articleID, locale, err)
	}
	if _, _, _, err := articleTranslationResourceURI(&mcp.ReadResourceRequest{Params: &mcp.ReadResourceParams{URI: "cms://articles/0/translations/zh-CN"}}); err == nil {
		t.Fatal("article resource accepted article ID zero")
	}
}

func TestCategoryResourceReadsTemplateURI(t *testing.T) {
	ctx := context.Background()
	categories := &categoryResourceFake{}
	server := New(Dependencies{Categories: categories})
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	uri := "cms://taxonomy/categories/zh-CN"
	result, err := clientSession.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
	if err != nil {
		t.Fatal(err)
	}
	if categories.locale != "zh-CN" || len(result.Contents) != 1 || result.Contents[0].URI != uri || result.Contents[0].Text != `{"data":[]}` {
		t.Fatalf("locale = %q, result = %#v", categories.locale, result)
	}
}

type categoryResourceFake struct {
	locale string
}

func (f *categoryResourceFake) Categories(_ context.Context, locale string) (json.RawMessage, error) {
	f.locale = locale
	return json.RawMessage(`{"data":[]}`), nil
}

func (*categoryResourceFake) CreateCategory(context.Context, contract.CategoryInput) (json.RawMessage, error) {
	return nil, nil
}

func (*categoryResourceFake) UpdateCategory(context.Context, uint, contract.CategoryStateInput) (json.RawMessage, error) {
	return nil, nil
}

func (*categoryResourceFake) MoveCategory(context.Context, uint, contract.CategoryMoveInput) (json.RawMessage, error) {
	return nil, nil
}

func (*categoryResourceFake) UpsertCategoryTranslation(context.Context, uint, string, contract.CategoryTranslationInput) (json.RawMessage, error) {
	return nil, nil
}

var _ contract.CategoryService = (*categoryResourceFake)(nil)

func TestOperationalInputValidation(t *testing.T) {
	if err := validateArticleReference(articleReferenceInput{ArticleID: 7, Locale: "zh-CN"}); err != nil {
		t.Fatalf("validateArticleReference() error = %v", err)
	}
	if err := validateArticleReference(articleReferenceInput{ArticleID: 7}); err == nil {
		t.Fatal("validateArticleReference() accepted missing locale")
	}
	if err := validateNamedTranslation("zh-CN", "Engineering", "engineering"); err != nil {
		t.Fatalf("validateNamedTranslation() error = %v", err)
	}
	if err := validateNamedTranslation("zh-CN", "", "engineering"); err == nil {
		t.Fatal("validateNamedTranslation() accepted missing name")
	}
	if err := validateLocaleInput("en-US", "English (United States)"); err != nil {
		t.Fatalf("validateLocaleInput() error = %v", err)
	}
	if err := validateLocaleInput("en_US", "English"); err == nil {
		t.Fatal("validateLocaleInput() accepted an invalid code")
	}
}

func TestOperationIDForUsesHostValueOrSessionFingerprint(t *testing.T) {
	input := articleIDInput{ArticleID: 7}
	if got := operationIDFor("session-1", "host-operation", "cms.article.restore", input); got != "host-operation" {
		t.Fatalf("host operation ID = %q", got)
	}
	first := operationIDFor("session-1", "", "cms.article.restore", input)
	second := operationIDFor("session-1", "", "cms.article.restore", input)
	if first != second {
		t.Fatalf("same session and input generated %q and %q", first, second)
	}
	if changed := operationIDFor("session-1", "", "cms.article.restore", articleIDInput{ArticleID: 8}); changed == first {
		t.Fatalf("different input generated identical operation ID %q", changed)
	}
}

func TestServerRegistersOperationalToolsAndPrompts(t *testing.T) {
	ctx := context.Background()
	server := New(Dependencies{})
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	wantTools := map[string]bool{
		"cms.article.create_translation":  false,
		"cms.article.archive":             false,
		"cms.article.restore":             false,
		"cms.article.set_cover":           false,
		"cms.category.create":             false,
		"cms.category.update":             false,
		"cms.category.move":               false,
		"cms.category.upsert_translation": false,
		"cms.tag.create":                  false,
		"cms.tag.upsert_translation":      false,
		"cms.locale.create":               false,
		"cms.locale.update":               false,
	}
	removedReadTools := map[string]bool{
		"cms.article.get_translation": false,
		"cms.category.list":           false,
	}
	for tool, err := range clientSession.Tools(ctx, nil) {
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := wantTools[tool.Name]; ok {
			wantTools[tool.Name] = true
		}
		if _, ok := removedReadTools[tool.Name]; ok {
			removedReadTools[tool.Name] = true
		}
	}
	for name, found := range wantTools {
		if !found {
			t.Errorf("tool %q was not registered", name)
		}
	}
	for name, found := range removedReadTools {
		if found {
			t.Errorf("read tool %q should be registered as a resource template", name)
		}
	}

	wantTemplates := map[string]bool{
		"cms://taxonomy/categories/{locale}":                false,
		"cms://articles/{article_id}/translations/{locale}": false,
	}
	for template, err := range clientSession.ResourceTemplates(ctx, nil) {
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := wantTemplates[template.URITemplate]; ok {
			wantTemplates[template.URITemplate] = true
		}
	}
	for uriTemplate, found := range wantTemplates {
		if !found {
			t.Errorf("resource template %q was not registered", uriTemplate)
		}
	}

	wantPrompts := map[string]bool{
		"cms.draft_from_brief":      false,
		"cms.pre_publish_review":    false,
		"cms.weekly_content_review": false,
	}
	for prompt, err := range clientSession.Prompts(ctx, nil) {
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := wantPrompts[prompt.Name]; ok {
			wantPrompts[prompt.Name] = true
		}
	}
	for name, found := range wantPrompts {
		if !found {
			t.Errorf("prompt %q was not registered", name)
		}
	}
	result, err := clientSession.GetPrompt(ctx, &mcp.GetPromptParams{Name: "cms.draft_from_brief", Arguments: map[string]string{"locale": "zh-CN", "brief": "Write about content operations"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Messages) != 1 || !strings.Contains(result.Messages[0].Content.(*mcp.TextContent).Text, "Write about content operations") {
		t.Fatalf("draft prompt = %#v", result)
	}
}
