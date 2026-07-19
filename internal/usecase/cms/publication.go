package cms

import (
	"context"
	"strings"

	domainCMS "github.com/freeDog-wy/go-backend-template/internal/domain/cms"
)

func (s *Service) PublishTranslation(ctx context.Context, cmd PublishTranslationCmd) (*ArticleResult, error) {
	var published *domainCMS.ArticleTranslation
	if err := s.tx.Do(ctx, func(ctx context.Context) error {
		result, err := s.evaluatePublication(ctx, cmd.ArticleID, cmd.Locale)
		if err != nil {
			return err
		}
		if !result.Publishable {
			return domainCMS.ErrPublicationNotReady
		}
		now := s.now().UTC()
		result.translation.Status, result.translation.PublishedAt = domainCMS.TranslationPublished, &now
		published = result.translation
		if err := s.repo.SaveArticleTranslation(ctx, published); err != nil {
			return err
		}
		return s.publishAudit(ctx, cmd.ActorUserID, "article_translation", published.ID, auditActionArticlePublished, cmd.IP, cmd.UserAgent, auditMetadata(map[string]any{"article_id": cmd.ArticleID, "locale": cmd.Locale}, cmd.CorrelationID))
	}); err != nil {
		return nil, err
	}
	return articleResult(cmd.ArticleID, published), nil
}

func (s *Service) PreviewPublish(ctx context.Context, cmd PreviewPublishCmd) (*PublishPreviewResult, error) {
	result, err := s.evaluatePublication(ctx, cmd.ArticleID, cmd.Locale)
	if err != nil {
		return nil, err
	}
	return &PublishPreviewResult{Publishable: result.Publishable, Article: articleResult(cmd.ArticleID, result.translation), Checks: result.Checks}, nil
}

type publicationEvaluation struct {
	translation *domainCMS.ArticleTranslation
	Publishable bool
	Checks      []PublishCheck
}

// evaluatePublication is the single source of truth for preview and publication.
func (s *Service) evaluatePublication(ctx context.Context, articleID uint, locale string) (*publicationEvaluation, error) {
	translation, err := s.translation(ctx, articleID, locale)
	if err != nil {
		return nil, err
	}
	article, err := s.repo.FindArticleIncludingDeleted(ctx, articleID)
	if err != nil {
		return nil, mapArticle(err)
	}
	categories, err := s.repo.ListArticleCategories(ctx, articleID)
	if err != nil {
		return nil, err
	}
	hasPrimaryCategory := false
	for _, category := range categories {
		if category.IsPrimary {
			hasPrimaryCategory = true
			break
		}
	}
	checks := []PublishCheck{
		{Name: "title", Passed: strings.TrimSpace(translation.Title) != "", Blocking: true, Message: "title is required"},
		{Name: "slug", Passed: strings.TrimSpace(translation.Slug) != "", Blocking: true, Message: "slug is required"},
		{Name: "content", Passed: strings.TrimSpace(translation.Content) != "", Blocking: true, Message: "content is required"},
		{Name: "content_format", Passed: translation.ContentFormat == "markdown" || translation.ContentFormat == "html", Blocking: true, Message: "content_format must be markdown or html"},
		{Name: "article_active", Passed: article.DeletedAt == nil, Blocking: true, Message: "article is deleted"},
		{Name: "seo_title", Passed: strings.TrimSpace(translation.SEOTitle) != "", Message: "SEO title is recommended"},
		{Name: "seo_description", Passed: strings.TrimSpace(translation.SEODescription) != "", Message: "SEO description is recommended"},
		{Name: "canonical_url", Passed: strings.TrimSpace(translation.CanonicalURL) != "", Message: "canonical URL is recommended"},
		{Name: "primary_category", Passed: hasPrimaryCategory, Message: "a primary category is recommended"},
	}
	publishable := true
	for _, check := range checks {
		if check.Blocking && !check.Passed {
			publishable = false
		}
	}
	return &publicationEvaluation{translation: translation, Publishable: publishable, Checks: checks}, nil
}
