package server

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/freeDog-wy/go-backend-template/internal/app/mcp/contract"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestContentLoaderLoadsUTF8FileUnderRoot(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "drafts", "article.md")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("# Draft\n"), 0600); err != nil {
		t.Fatal(err)
	}

	content, digest, err := newContentLoader(root).load("drafts/article.md")
	if err != nil {
		t.Fatal(err)
	}
	if content != "# Draft\n" || digest != "c47fffce7ab6215da4633829b59605e9bdf14fb3d49b6ac0fe8105e639b9c4f9" {
		t.Fatalf("load() = (%q, %q), want content and digest", content, digest)
	}
}

func TestContentLoaderRejectsUnsafePathsAndInvalidContent(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "invalid.md"), []byte{0xff}, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "directory"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "outside-link.md")); err != nil {
		t.Skipf("create symlink: %v", err)
	}

	loader := newContentLoader(root)
	for _, name := range []string{"../outside.md", outside, "outside-link.md", "invalid.md", "directory"} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := loader.load(name); err == nil {
				t.Fatalf("load(%q) error = nil", name)
			}
		})
	}
	if _, _, err := newContentLoader("").load("draft.md"); err == nil || !strings.Contains(err.Error(), "CMS_CONTENT_ROOT") {
		t.Fatalf("empty root error = %v", err)
	}
}

func TestResolvedFileContentChangesWriteOperationID(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "article.md")
	if err := os.WriteFile(path, []byte("first"), 0600); err != nil {
		t.Fatal(err)
	}
	input := articleWriteInput{Locale: "zh-CN", Title: "Draft", Slug: "draft", ContentFile: "article.md"}
	first, err := resolveArticleWrite(input, newContentLoader(root))
	if err != nil {
		t.Fatal(err)
	}
	if first.input.Content != "first" {
		t.Fatalf("resolved content = %q", first.input.Content)
	}
	if input := articleInput(first.input); input.Content != "first" {
		t.Fatalf("CMS article input content = %q", input.Content)
	}
	firstOperationID := operationIDFor("session-1", "", "cms.article.create_draft", first.operationInput())

	if err := os.WriteFile(path, []byte("second"), 0600); err != nil {
		t.Fatal(err)
	}
	second, err := resolveArticleWrite(input, newContentLoader(root))
	if err != nil {
		t.Fatal(err)
	}
	secondOperationID := operationIDFor("session-1", "", "cms.article.create_draft", second.operationInput())
	if firstOperationID == secondOperationID {
		t.Fatalf("changed file content generated identical operation ID %q", firstOperationID)
	}
}

func TestArticleCreateDraftLoadsContentFileBeforeCallingCMS(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "article.md"), []byte("# File-backed draft\n"), 0600); err != nil {
		t.Fatal(err)
	}
	articles := &articleWriteFake{}
	server := New(Dependencies{Articles: articles, ContentRoot: root})
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

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "cms.article.create_draft", Arguments: map[string]any{
		"locale": "zh-CN", "title": "File-backed", "slug": "file-backed", "content_file": "article.md",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError || articles.input.Content != "# File-backed draft\n" || articles.operationID == "" {
		t.Fatalf("result = %#v, CMS input = %#v, operation ID = %q", result, articles.input, articles.operationID)
	}
}

func TestValidateArticleInputRejectsInlineAndFileContent(t *testing.T) {
	err := validateArticleInput(articleWriteInput{Locale: "zh-CN", Title: "Draft", Slug: "draft", Content: "inline", ContentFile: "draft.md"}, false)
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("validateArticleInput() error = %v", err)
	}
}

type articleWriteFake struct {
	input       contract.ArticleInput
	operationID string
}

func (f *articleWriteFake) CreateArticleDraft(ctx context.Context, input contract.ArticleInput) (json.RawMessage, error) {
	f.input = input
	f.operationID = contract.WriteOperationID(ctx)
	return json.RawMessage(`{"id":7}`), nil
}

func (*articleWriteFake) Articles(context.Context, string, string, int, int) (json.RawMessage, error) {
	return nil, nil
}

func (*articleWriteFake) ArticleTranslation(context.Context, uint, string) (json.RawMessage, error) {
	return nil, nil
}

func (*articleWriteFake) CreateArticleTranslation(context.Context, uint, contract.ArticleInput) (json.RawMessage, error) {
	return nil, nil
}

func (*articleWriteFake) UpdateArticleTranslation(context.Context, uint, string, contract.ArticleInput) (json.RawMessage, error) {
	return nil, nil
}

func (*articleWriteFake) ReplaceArticleCategories(context.Context, uint, []uint, *uint) (json.RawMessage, error) {
	return nil, nil
}

func (*articleWriteFake) ReplaceArticleTags(context.Context, uint, []uint) (json.RawMessage, error) {
	return nil, nil
}

func (*articleWriteFake) PreviewPublish(context.Context, uint, string) (json.RawMessage, error) {
	return nil, nil
}

func (*articleWriteFake) PublishArticleTranslation(context.Context, uint, string) (json.RawMessage, error) {
	return nil, nil
}

func (*articleWriteFake) ArchiveArticleTranslation(context.Context, uint, string) (json.RawMessage, error) {
	return nil, nil
}

func (*articleWriteFake) RestoreArticle(context.Context, uint) (json.RawMessage, error) {
	return nil, nil
}

func (*articleWriteFake) SetArticleCover(context.Context, uint, *uint) (json.RawMessage, error) {
	return nil, nil
}

var _ contract.ArticleService = (*articleWriteFake)(nil)
