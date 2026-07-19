package cms

import (
	"context"
	"errors"
	"strconv"
	"strings"
	
	domainCMS "github.com/freeDog-wy/go-backend-template/internal/domain/cms"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
	platformAudit "github.com/freeDog-wy/go-backend-template/internal/platform/audit"
)

func (s *Service) translation(ctx context.Context, id uint, locale string) (*domainCMS.ArticleTranslation, error) {
	if id == 0 || strings.TrimSpace(locale) == "" {
		return nil, domainCMS.ErrInvalidInput
	}
	tr, err := s.repo.FindArticleTranslation(ctx, id, locale)
	if errors.Is(err, shared.ErrNotFound) {
		return nil, domainCMS.ErrTranslationAbsent
	}
	return tr, err
}
func (s *Service) requireLocale(ctx context.Context, locale string) error {
	ok, err := s.repo.LocaleEnabled(ctx, strings.TrimSpace(locale))
	if err != nil {
		return err
	}
	if !ok {
		return domainCMS.ErrLocaleNotFound
	}
	return nil
}
func (s *Service) requireExistingLocale(ctx context.Context, locale string) error {
	_, err := s.repo.FindLocale(ctx, strings.TrimSpace(locale))
	return mapLocale(err)
}
func validNameSlug(name, slug string) error {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(slug) == "" {
		return domainCMS.ErrInvalidInput
	}
	return nil
}
func validArticle(title, slug, format string) error {
	if err := validNameSlug(title, slug); err != nil {
		return err
	}
	if format != "" && format != "markdown" && format != "html" {
		return domainCMS.ErrInvalidInput
	}
	return nil
}
func mapCategory(err error) error {
	if errors.Is(err, shared.ErrNotFound) {
		return domainCMS.ErrCategoryNotFound
	}
	return err
}
func mapArticle(err error) error {
	if errors.Is(err, shared.ErrNotFound) {
		return domainCMS.ErrArticleNotFound
	}
	return err
}
func mapTag(err error) error {
	if errors.Is(err, shared.ErrNotFound) {
		return domainCMS.ErrTagNotFound
	}
	return err
}
func mapLocale(err error) error {
	if errors.Is(err, shared.ErrNotFound) {
		return domainCMS.ErrLocaleNotFound
	}
	return err
}
func validLocale(code string) bool {
	if len(code) < 2 || len(code) > 35 {
		return false
	}
	for _, r := range code {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-') {
			return false
		}
	}
	return true
}
func (s *Service) publishAudit(ctx context.Context, actorUserID uint, targetType string, targetID uint, action, ip, userAgent string, metadata map[string]any) error {
	return s.publishAuditText(ctx, actorUserID, targetType, strconv.FormatUint(uint64(targetID), 10), action, ip, userAgent, metadata)
}
func (s *Service) publishAuditText(ctx context.Context, actorUserID uint, targetType, targetID, action, ip, userAgent string, metadata map[string]any) error {
	if s.auditor == nil {
		return nil
	}
	var actor *uint
	if actorUserID != 0 {
		actor = &actorUserID
	}
	return s.auditor.Record(ctx, platformAudit.RecordInput{ActorUserID: actor, TargetType: targetType, TargetID: targetID, Action: action, Result: platformAudit.ResultSuccess, IP: ip, UserAgent: userAgent, Metadata: metadata})
}
func auditMetadata(metadata map[string]any, correlationID string) map[string]any {
	if strings.TrimSpace(correlationID) != "" {
		metadata["correlation_id"] = correlationID
	}
	return metadata
}

func tagResult(id uint, tr *domainCMS.TagTranslation) *TagResult {
	return &TagResult{ID: id, Locale: tr.Locale, Name: tr.Name, Slug: tr.Slug}
}
func localeResult(locale *domainCMS.Locale) *LocaleResult {
	return &LocaleResult{Code: locale.Code, Name: locale.Name, IsDefault: locale.IsDefault, IsEnabled: locale.IsEnabled, SortOrder: locale.SortOrder}
}
func translationFromCreate(articleID uint, cmd CreateArticleCmd) *domainCMS.ArticleTranslation {
	format := cmd.ContentFormat
	if format == "" {
		format = "markdown"
	}
	return &domainCMS.ArticleTranslation{ArticleID: articleID, Locale: cmd.Locale, Title: cmd.Title, Slug: cmd.Slug, Summary: cmd.Summary, Content: cmd.Content, ContentFormat: format, Status: domainCMS.TranslationDraft, SEOTitle: cmd.SEOTitle, SEODescription: cmd.SEODescription, CanonicalURL: cmd.CanonicalURL}
}
func articleResult(id uint, tr *domainCMS.ArticleTranslation) *ArticleResult {
	return &ArticleResult{ID: id, Locale: tr.Locale, Title: tr.Title, Slug: tr.Slug, Status: string(tr.Status), PublishedAt: tr.PublishedAt}
}

