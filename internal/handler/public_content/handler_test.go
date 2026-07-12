package public_content

import (
	"context"
	"fmt"
	domainCMS "github.com/freeDog-wy/go-backend-template/internal/domain/cms"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
	svcCMS "github.com/freeDog-wy/go-backend-template/internal/usecase/cms"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type contentStub struct {
	result    *svcCMS.PublicArticleResult
	locales   []*svcCMS.LocaleResult
	tags      []*svcCMS.TagResult
	redirects []*svcCMS.RedirectResult
	err       error
}

func (s contentStub) ListPublishedLocales(context.Context) ([]*svcCMS.LocaleResult, error) {
	return s.locales, s.err
}
func (s contentStub) GetPublishedArticle(context.Context, string, string) (*svcCMS.PublicArticleResult, error) {
	return s.result, s.err
}
func (s contentStub) ListPublishedArticles(context.Context, svcCMS.ListPublicArticlesCmd) ([]*svcCMS.PublicArticleListResult, shared.PageResult, error) {
	return nil, shared.PageResult{}, s.err
}
func (s contentStub) ListPublishedCategoryArticles(context.Context, svcCMS.ListPublicCategoryArticlesCmd) ([]*svcCMS.PublicArticleListResult, shared.PageResult, error) {
	return nil, shared.PageResult{}, s.err
}
func (s contentStub) ListPublishedCategories(context.Context, string) ([]*svcCMS.CategoryTreeResult, error) {
	return nil, s.err
}
func (s contentStub) ListPublicSitemapEntries(context.Context, svcCMS.ListPublicSitemapEntriesCmd) ([]*svcCMS.SitemapEntryResult, shared.PageResult, error) {
	return nil, shared.PageResult{}, s.err
}
func (s contentStub) ResolveRedirect(context.Context, string, string) (*svcCMS.RedirectResult, error) {
	return nil, s.err
}
func (s contentStub) ListPublicRedirects(context.Context, svcCMS.ListPublicRedirectsCmd) ([]*svcCMS.RedirectResult, shared.PageResult, error) {
	return s.redirects, shared.PageResult{Page: 1, PerPage: 20, Total: int64(len(s.redirects))}, s.err
}
func (s contentStub) ListPublishedTags(context.Context, svcCMS.ListPublicTagsCmd) ([]*svcCMS.TagResult, shared.PageResult, error) {
	return s.tags, shared.PageResult{Page: 1, PerPage: 20, Total: int64(len(s.tags))}, s.err
}
func (s contentStub) ListPublishedTagArticles(context.Context, svcCMS.ListPublicTagArticlesCmd) ([]*svcCMS.PublicArticleListResult, shared.PageResult, error) {
	return nil, shared.PageResult{}, s.err
}
func TestGetArticleReturnsBusinessNotFoundWithHTTP200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	New(contentStub{err: fmt.Errorf("wrapped: %w", domainCMS.ErrTranslationAbsent)}).RegisterRoutes(r)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/public/en-US/articles/missing", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "CONTENT_TRANSLATION_NOT_FOUND") {
		t.Fatalf("response = %d %s", w.Code, w.Body.String())
	}
}

func TestStaticBuildDiscoveryRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	New(contentStub{
		locales:   []*svcCMS.LocaleResult{{Code: "zh-CN", IsEnabled: true, IsDefault: true}},
		tags:      []*svcCMS.TagResult{{ID: 1, Locale: "zh-CN", Name: "Go", Slug: "go"}},
		redirects: []*svcCMS.RedirectResult{{SourcePath: "/zh-CN/articles/old", TargetPath: "/zh-CN/articles/new", StatusCode: 301}},
	}).RegisterRoutes(r)
	for _, path := range []string{
		"/api/v1/public/locales",
		"/api/v1/public/zh-CN/tags?page=1&per_page=10",
		"/api/v1/public/zh-CN/redirects?page=1&per_page=10",
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"success":true`) {
			t.Fatalf("path = %s, response = %d %s", path, w.Code, w.Body.String())
		}
	}
}
