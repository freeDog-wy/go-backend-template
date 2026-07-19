package cms

import (
	"context"
	"errors"
	"fmt"
	"strings"
	
	domainCMS "github.com/freeDog-wy/go-backend-template/internal/domain/cms"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
)

func (s *Service) GetPublishedArticle(ctx context.Context, locale, slug string) (*PublicArticleResult, error) {
	if err := s.requireLocale(ctx, locale); err != nil {
		return nil, err
	}
	a, err := s.repo.FindPublicArticle(ctx, locale, slug)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, domainCMS.ErrTranslationAbsent
		}
		return nil, err
	}
	locales, err := s.repo.ListPublishedArticleLocales(ctx, a.Article.ID)
	if err != nil {
		return nil, err
	}
	breadcrumbs, err := s.repo.ListPublicArticleBreadcrumbs(ctx, a.Article.ID, locale)
	if err != nil {
		return nil, err
	}
	covers, err := s.publicCovers(ctx, locale, articleCoverIDs([]*domainCMS.PublicArticleListItem{{Article: a.Article}}))
	if err != nil {
		return nil, err
	}
	result := &PublicArticleResult{ID: a.Article.ID, Locale: a.Locale, Title: a.Title, Slug: a.Slug, Summary: a.Summary, Content: a.Content, ContentFormat: a.ContentFormat, PublishedAt: a.PublishedAt, SEOTitle: a.SEOTitle, SEODescription: a.SEODescription, CanonicalURL: a.CanonicalURL, Cover: coverFor(a.Article.CoverMediaID, covers), UpdatedAt: a.ArticleTranslation.UpdatedAt, AvailableLocales: make([]PublicLocaleRef, 0, len(locales)), Breadcrumbs: make([]PublicCategoryRef, 0, len(breadcrumbs))}
	for _, translation := range locales {
		result.AvailableLocales = append(result.AvailableLocales, PublicLocaleRef{Locale: translation.Locale, Slug: translation.Slug})
	}
	for index, category := range breadcrumbs {
		ref := PublicCategoryRef{ID: category.ID, Name: category.Name, Slug: category.Slug}
		result.Breadcrumbs = append(result.Breadcrumbs, ref)
		if index == len(breadcrumbs)-1 {
			result.PrimaryCategory = &ref
		}
	}
	return result, nil
}
func (s *Service) ListPublicSitemapEntries(ctx context.Context, cmd ListPublicSitemapEntriesCmd) ([]*SitemapEntryResult, shared.PageResult, error) {
	if err := s.requireLocale(ctx, cmd.Locale); err != nil {
		return nil, shared.PageResult{}, err
	}
	page := shared.NewPageQuery(cmd.Page.Page, cmd.Page.PerPage)
	entries, total, err := s.repo.ListPublicSitemapEntries(ctx, cmd.Locale, page)
	if err != nil {
		return nil, shared.PageResult{}, err
	}
	results := make([]*SitemapEntryResult, 0, len(entries))
	for _, entry := range entries {
		var path string
		switch entry.Kind {
		case "article":
			path = fmt.Sprintf("/%s/articles/%s", cmd.Locale, entry.Slug)
		case "category":
			path = fmt.Sprintf("/%s/categories/%s", cmd.Locale, entry.Slug)
		default:
			continue
		}
		results = append(results, &SitemapEntryResult{URL: path, LastModified: entry.UpdatedAt})
	}
	return results, shared.PageResult{Page: page.Page, PerPage: page.PerPage, Total: total}, nil
}
func (s *Service) ListPublishedArticles(ctx context.Context, cmd ListPublicArticlesCmd) ([]*PublicArticleListResult, shared.PageResult, error) {
	return s.listPublishedArticles(ctx, cmd.Locale, nil, cmd.Page)
}
func (s *Service) ListPublishedCategoryArticles(ctx context.Context, cmd ListPublicCategoryArticlesCmd) ([]*PublicArticleListResult, shared.PageResult, error) {
	if strings.TrimSpace(cmd.CategorySlug) == "" {
		return nil, shared.PageResult{}, domainCMS.ErrInvalidInput
	}
	if err := s.requireLocale(ctx, cmd.Locale); err != nil {
		return nil, shared.PageResult{}, err
	}
	exists, err := s.repo.PublicCategoryExists(ctx, cmd.Locale, cmd.CategorySlug)
	if err != nil {
		return nil, shared.PageResult{}, err
	}
	if !exists {
		return nil, shared.PageResult{}, domainCMS.ErrCategoryNotFound
	}
	return s.listPublishedArticles(ctx, cmd.Locale, &cmd.CategorySlug, cmd.Page)
}
func (s *Service) ListPublishedTagArticles(ctx context.Context, cmd ListPublicTagArticlesCmd) ([]*PublicArticleListResult, shared.PageResult, error) {
	if strings.TrimSpace(cmd.TagSlug) == "" {
		return nil, shared.PageResult{}, domainCMS.ErrInvalidInput
	}
	if err := s.requireLocale(ctx, cmd.Locale); err != nil {
		return nil, shared.PageResult{}, err
	}
	ok, err := s.repo.PublicTagExists(ctx, cmd.Locale, cmd.TagSlug)
	if err != nil {
		return nil, shared.PageResult{}, err
	}
	if !ok {
		return nil, shared.PageResult{}, domainCMS.ErrTagNotFound
	}
	page := shared.NewPageQuery(cmd.Page.Page, cmd.Page.PerPage)
	items, total, err := s.repo.ListPublicTagArticles(ctx, cmd.Locale, cmd.TagSlug, page)
	if err != nil {
		return nil, shared.PageResult{}, err
	}
	covers, err := s.publicCovers(ctx, cmd.Locale, articleCoverIDs(items))
	if err != nil {
		return nil, shared.PageResult{}, err
	}
	results := make([]*PublicArticleListResult, 0, len(items))
	for _, item := range items {
		results = append(results, &PublicArticleListResult{ID: item.Article.ID, Locale: item.Locale, Title: item.Title, Slug: item.Slug, Summary: item.Summary, ContentFormat: item.ContentFormat, PublishedAt: item.PublishedAt, Cover: coverFor(item.Article.CoverMediaID, covers), UpdatedAt: item.ArticleTranslation.UpdatedAt})
	}
	return results, shared.PageResult{Page: page.Page, PerPage: page.PerPage, Total: total}, nil
}
func (s *Service) listPublishedArticles(ctx context.Context, locale string, categorySlug *string, page shared.PageQuery) ([]*PublicArticleListResult, shared.PageResult, error) {
	if err := s.requireLocale(ctx, locale); err != nil {
		return nil, shared.PageResult{}, err
	}
	page = shared.NewPageQuery(page.Page, page.PerPage)
	items, total, err := s.repo.ListPublicArticles(ctx, locale, categorySlug, page)
	if err != nil {
		return nil, shared.PageResult{}, err
	}
	covers, err := s.publicCovers(ctx, locale, articleCoverIDs(items))
	if err != nil {
		return nil, shared.PageResult{}, err
	}
	results := make([]*PublicArticleListResult, 0, len(items))
	for _, item := range items {
		result := &PublicArticleListResult{ID: item.Article.ID, Locale: item.Locale, Title: item.Title, Slug: item.Slug, Summary: item.Summary, ContentFormat: item.ContentFormat, PublishedAt: item.PublishedAt, Cover: coverFor(item.Article.CoverMediaID, covers), UpdatedAt: item.ArticleTranslation.UpdatedAt}
		if item.PrimaryCategoryID != nil {
			result.PrimaryCategory = &PublicCategoryRef{ID: *item.PrimaryCategoryID, Name: item.PrimaryCategoryName, Slug: item.PrimaryCategorySlug}
		}
		results = append(results, result)
	}
	return results, shared.PageResult{Page: page.Page, PerPage: page.PerPage, Total: total}, nil
}
func articleCoverIDs(items []*domainCMS.PublicArticleListItem) []uint {
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		if item.Article.CoverMediaID != nil {
			ids = append(ids, *item.Article.CoverMediaID)
		}
	}
	return ids
}

func (s *Service) publicCovers(ctx context.Context, locale string, ids []uint) (map[uint]*CoverMediaResult, error) {
	result := make(map[uint]*CoverMediaResult)
	if s.publicMediaFinder == nil {
		return result, nil
	}
	assets, err := s.publicMediaFinder.ListPublic(ctx, locale, ids)
	if err != nil {
		return nil, err
	}
	for _, asset := range assets {
		result[asset.ID] = &CoverMediaResult{ID: asset.ID, URL: asset.URL, AltText: asset.AltText, Title: asset.Title}
	}
	return result, nil
}

func coverFor(id *uint, covers map[uint]*CoverMediaResult) *CoverMediaResult {
	if id == nil {
		return nil
	}
	return covers[*id]
}

