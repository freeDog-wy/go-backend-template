package cms

import (
	"context"
	"errors"
	"fmt"
	"strings"
	
	domainCMS "github.com/freeDog-wy/go-backend-template/internal/domain/cms"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
)

func (s *Service) ResolveRedirect(ctx context.Context, locale, sourcePath string) (*RedirectResult, error) {
	if err := s.requireLocale(ctx, locale); err != nil {
		return nil, err
	}
	if !strings.HasPrefix(sourcePath, "/"+locale+"/") {
		return nil, domainCMS.ErrInvalidInput
	}
	redirect, err := s.repo.FindURLRedirect(ctx, locale, sourcePath)
	if errors.Is(err, shared.ErrNotFound) {
		return nil, domainCMS.ErrRedirectNotFound
	}
	if err != nil {
		return nil, err
	}
	return &RedirectResult{SourcePath: redirect.SourcePath, TargetPath: redirect.TargetPath, StatusCode: redirect.StatusCode}, nil
}
func (s *Service) ListPublicRedirects(ctx context.Context, cmd ListPublicRedirectsCmd) ([]*RedirectResult, shared.PageResult, error) {
	if err := s.requireLocale(ctx, cmd.Locale); err != nil {
		return nil, shared.PageResult{}, err
	}
	page := shared.NewPageQuery(cmd.Page.Page, cmd.Page.PerPage)
	redirects, total, err := s.repo.ListURLRedirects(ctx, cmd.Locale, page)
	if err != nil {
		return nil, shared.PageResult{}, err
	}
	result := make([]*RedirectResult, 0, len(redirects))
	for _, redirect := range redirects {
		result = append(result, &RedirectResult{SourcePath: redirect.SourcePath, TargetPath: redirect.TargetPath, StatusCode: redirect.StatusCode})
	}
	return result, shared.PageResult{Page: page.Page, PerPage: page.PerPage, Total: total}, nil
}

func (s *Service) ensureSlugAvailable(ctx context.Context, locale, path string) error {
	exists, err := s.repo.RedirectSourceExists(ctx, locale, path)
	if err != nil {
		return err
	}
	if exists {
		return domainCMS.ErrSlugReserved
	}
	return nil
}
func articlePath(locale, slug string) string  { return fmt.Sprintf("/%s/articles/%s", locale, slug) }
func categoryPath(locale, slug string) string { return fmt.Sprintf("/%s/categories/%s", locale, slug) }
func tagPath(locale, slug string) string      { return fmt.Sprintf("/%s/tags/%s", locale, slug) }

