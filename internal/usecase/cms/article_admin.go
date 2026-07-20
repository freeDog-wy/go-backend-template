package cms

import (
	"context"
	"fmt"

	domainCMS "github.com/freeDog-wy/go-backend-template/internal/domain/cms"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
)

func (s *Service) ReplaceArticleTags(ctx context.Context, cmd ReplaceArticleTagsCmd) error {
	if cmd.ArticleID == 0 {
		return domainCMS.ErrInvalidInput
	}
	if _, err := s.repo.FindArticle(ctx, cmd.ArticleID); err != nil {
		return mapArticle(err)
	}
	seen := map[uint]struct{}{}
	for _, id := range cmd.TagIDs {
		if id == 0 {
			return domainCMS.ErrInvalidInput
		}
		if _, ok := seen[id]; ok {
			return domainCMS.ErrInvalidInput
		}
		seen[id] = struct{}{}
		if _, err := s.repo.FindTag(ctx, id); err != nil {
			return mapTag(err)
		}
	}
	return s.tx.Do(ctx, func(ctx context.Context) error {
		if err := s.repo.ReplaceArticleTags(ctx, cmd.ArticleID, cmd.TagIDs); err != nil {
			return err
		}
		return s.publishAudit(ctx, cmd.ActorUserID, "article", cmd.ArticleID, auditActionArticleTagsChanged, cmd.IP, cmd.UserAgent, auditMetadata(map[string]any{"tag_ids": cmd.TagIDs}, cmd.CorrelationID))
	})
}

func (s *Service) CreateArticle(ctx context.Context, cmd CreateArticleCmd) (*ArticleResult, error) {
	if cmd.AuthorUserID == 0 {
		return nil, domainCMS.ErrInvalidInput
	}
	if err := s.requireLocale(ctx, cmd.Locale); err != nil {
		return nil, err
	}
	if err := validArticle(cmd.Title, cmd.Slug, cmd.ContentFormat); err != nil {
		return nil, err
	}
	a := &domainCMS.Article{AuthorUserID: cmd.AuthorUserID}
	tr := translationFromCreate(0, cmd)
	if err := s.tx.Do(ctx, func(ctx context.Context) error {
		if err := s.repo.CreateArticle(ctx, a, tr); err != nil {
			return err
		}
		return s.publishAudit(ctx, cmd.AuthorUserID, "article", a.ID, auditActionArticleCreated, cmd.IP, cmd.UserAgent, auditMetadata(map[string]any{"locale": tr.Locale, "slug": tr.Slug}, cmd.CorrelationID))
	}); err != nil {
		return nil, err
	}
	return articleResult(a.ID, tr), nil
}
func (s *Service) CreateTranslation(ctx context.Context, cmd CreateTranslationCmd) (*ArticleResult, error) {
	if cmd.ArticleID == 0 {
		return nil, domainCMS.ErrInvalidInput
	}
	if err := s.requireLocale(ctx, cmd.Locale); err != nil {
		return nil, err
	}
	if err := validArticle(cmd.Title, cmd.Slug, cmd.ContentFormat); err != nil {
		return nil, err
	}
	tr := translationFromCreate(cmd.ArticleID, CreateArticleCmd{Locale: cmd.Locale, Title: cmd.Title, Slug: cmd.Slug, Summary: cmd.Summary, Content: cmd.Content, ContentFormat: cmd.ContentFormat, SEOTitle: cmd.SEOTitle, SEODescription: cmd.SEODescription, CanonicalURL: cmd.CanonicalURL})
	if err := s.repo.CreateArticleTranslation(ctx, tr); err != nil {
		return nil, err
	}
	return articleResult(cmd.ArticleID, tr), nil
}
func (s *Service) UpdateTranslation(ctx context.Context, cmd UpdateTranslationCmd) (*ArticleResult, error) {
	if err := validArticle(cmd.Title, cmd.Slug, cmd.ContentFormat); err != nil {
		return nil, err
	}
	tr, err := s.translation(ctx, cmd.ArticleID, cmd.Locale)
	if err != nil {
		return nil, err
	}
	oldSlug := tr.Slug
	wasPublic := tr.Status == domainCMS.TranslationPublished && tr.PublishedAt != nil && !tr.PublishedAt.After(s.now())
	tr.Title, tr.Slug, tr.Summary, tr.Content, tr.ContentFormat, tr.SEOTitle, tr.SEODescription, tr.CanonicalURL = cmd.Title, cmd.Slug, cmd.Summary, cmd.Content, cmd.ContentFormat, cmd.SEOTitle, cmd.SEODescription, cmd.CanonicalURL
	if err := s.tx.Do(ctx, func(ctx context.Context) error {
		if oldSlug != tr.Slug && wasPublic {
			if err := s.ensureSlugAvailable(ctx, tr.Locale, articlePath(tr.Locale, tr.Slug)); err != nil {
				return err
			}
		}
		if err := s.repo.SaveArticleTranslation(ctx, tr); err != nil {
			return err
		}
		if oldSlug != tr.Slug && wasPublic {
			redirect := &domainCMS.URLRedirect{Locale: tr.Locale, SourcePath: articlePath(tr.Locale, oldSlug), TargetPath: articlePath(tr.Locale, tr.Slug), StatusCode: 301}
			if err := s.repo.SaveURLRedirect(ctx, redirect); err != nil {
				return err
			}
			return s.publishAudit(ctx, cmd.ActorUserID, "article_translation", tr.ID, auditActionSlugChanged, cmd.IP, cmd.UserAgent, auditMetadata(map[string]any{"article_id": cmd.ArticleID, "locale": tr.Locale, "old_slug": oldSlug, "new_slug": tr.Slug}, cmd.CorrelationID))
		}
		return s.publishAudit(ctx, cmd.ActorUserID, "article_translation", tr.ID, auditActionArticleUpdated, cmd.IP, cmd.UserAgent, auditMetadata(map[string]any{"article_id": cmd.ArticleID, "locale": tr.Locale}, cmd.CorrelationID))
	}); err != nil {
		return nil, err
	}
	return articleResult(cmd.ArticleID, tr), nil
}

func (s *Service) ArchiveTranslation(ctx context.Context, cmd ArchiveTranslationCmd) (*ArticleResult, error) {
	tr, err := s.translation(ctx, cmd.ArticleID, cmd.Locale)
	if err != nil {
		return nil, err
	}
	tr.Status = domainCMS.TranslationArchived
	if err := s.tx.Do(ctx, func(ctx context.Context) error {
		if err := s.repo.SaveArticleTranslation(ctx, tr); err != nil {
			return err
		}
		return s.publishAudit(ctx, cmd.ActorUserID, "article_translation", tr.ID, auditActionArticleArchived, cmd.IP, cmd.UserAgent, map[string]any{"article_id": cmd.ArticleID, "locale": cmd.Locale})
	}); err != nil {
		return nil, err
	}
	return articleResult(cmd.ArticleID, tr), nil
}
func (s *Service) DeleteArticle(ctx context.Context, cmd DeleteArticleCmd) error {
	if cmd.ArticleID == 0 {
		return domainCMS.ErrInvalidInput
	}
	article, err := s.repo.FindArticleIncludingDeleted(ctx, cmd.ArticleID)
	if err != nil {
		return mapArticle(err)
	}
	if article.DeletedAt != nil {
		return domainCMS.ErrArticleDeleted
	}
	now := s.now().UTC()
	return s.tx.Do(ctx, func(ctx context.Context) error {
		if err := s.repo.SoftDeleteArticle(ctx, cmd.ArticleID, now); err != nil {
			return err
		}
		return s.publishAudit(ctx, cmd.ActorUserID, "article", cmd.ArticleID, auditActionArticleDeleted, cmd.IP, cmd.UserAgent, map[string]any{"deleted_at": now})
	})
}
func (s *Service) RestoreArticle(ctx context.Context, cmd RestoreArticleCmd) error {
	if cmd.ArticleID == 0 {
		return domainCMS.ErrInvalidInput
	}
	article, err := s.repo.FindArticleIncludingDeleted(ctx, cmd.ArticleID)
	if err != nil {
		return mapArticle(err)
	}
	if article.DeletedAt == nil {
		return domainCMS.ErrArticleActive
	}
	return s.tx.Do(ctx, func(ctx context.Context) error {
		if err := s.repo.RestoreArticle(ctx, cmd.ArticleID); err != nil {
			return err
		}
		return s.publishAudit(ctx, cmd.ActorUserID, "article", cmd.ArticleID, auditActionArticleRestored, cmd.IP, cmd.UserAgent, map[string]any{"deleted_at": article.DeletedAt})
	})
}
func (s *Service) SetArticleCover(ctx context.Context, cmd SetArticleCoverCmd) error {
	if cmd.ArticleID == 0 {
		return domainCMS.ErrInvalidInput
	}
	if _, err := s.repo.FindArticle(ctx, cmd.ArticleID); err != nil {
		return mapArticle(err)
	}
	if cmd.MediaID != nil {
		if s.mediaFinder == nil {
			return fmt.Errorf("media service is not configured")
		}
		ok, err := s.mediaFinder.IsReady(ctx, *cmd.MediaID)
		if err != nil || !ok {
			return fmt.Errorf("media is not ready")
		}
	}
	return s.tx.Do(ctx, func(ctx context.Context) error {
		if err := s.repo.SetArticleCover(ctx, cmd.ArticleID, cmd.MediaID); err != nil {
			return err
		}
		return s.publishAudit(ctx, cmd.ActorUserID, "article", cmd.ArticleID, auditActionArticleCoverChanged, cmd.IP, cmd.UserAgent, map[string]any{"cover_media_id": cmd.MediaID})
	})
}

func (s *Service) ReplaceArticleCategories(ctx context.Context, cmd ReplaceArticleCategoriesCmd) error {
	if cmd.ArticleID == 0 {
		return domainCMS.ErrInvalidInput
	}
	if _, err := s.repo.FindArticle(ctx, cmd.ArticleID); err != nil {
		return mapArticle(err)
	}
	ids := make([]uint, 0, len(cmd.CategoryIDs))
	seen := make(map[uint]struct{}, len(cmd.CategoryIDs))
	for _, id := range cmd.CategoryIDs {
		if id == 0 {
			return domainCMS.ErrInvalidInput
		}
		if _, ok := seen[id]; ok {
			return domainCMS.ErrInvalidInput
		}
		seen[id] = struct{}{}
		if _, err := s.repo.FindCategory(ctx, id); err != nil {
			return mapCategory(err)
		}
		ids = append(ids, id)
	}
	if cmd.PrimaryCategoryID != nil {
		if _, ok := seen[*cmd.PrimaryCategoryID]; !ok {
			return domainCMS.ErrInvalidInput
		}
	}
	return s.tx.Do(ctx, func(ctx context.Context) error {
		if err := s.repo.ReplaceArticleCategories(ctx, cmd.ArticleID, ids, cmd.PrimaryCategoryID); err != nil {
			return err
		}
		return s.publishAudit(ctx, cmd.ActorUserID, "article", cmd.ArticleID, auditActionArticleCategoriesChanged, cmd.IP, cmd.UserAgent, auditMetadata(map[string]any{"category_ids": ids, "primary_category_id": cmd.PrimaryCategoryID}, cmd.CorrelationID))
	})
}
func (s *Service) ListArticles(ctx context.Context, cmd ListArticlesCmd) ([]*ArticleResult, shared.PageResult, error) {
	if err := s.requireLocale(ctx, cmd.Locale); err != nil {
		return nil, shared.PageResult{}, err
	}
	if cmd.Status != "" && cmd.Status != domainCMS.TranslationDraft && cmd.Status != domainCMS.TranslationPublished && cmd.Status != domainCMS.TranslationArchived {
		return nil, shared.PageResult{}, domainCMS.ErrInvalidInput
	}
	page := shared.NewPageQuery(cmd.Page.Page, cmd.Page.PerPage)
	items, total, err := s.repo.ListArticleTranslations(ctx, cmd.Locale, cmd.Status, cmd.IncludeDeleted, page)
	if err != nil {
		return nil, shared.PageResult{}, err
	}
	results := make([]*ArticleResult, 0, len(items))
	for _, item := range items {
		results = append(results, articleResult(item.Article.ID, &item.ArticleTranslation))
	}
	return results, shared.PageResult{Page: page.Page, PerPage: page.PerPage, Total: total}, nil
}
func (s *Service) GetArticleTranslation(ctx context.Context, cmd GetArticleTranslationCmd) (*ArticleDetailResult, error) {
	if cmd.ArticleID == 0 {
		return nil, domainCMS.ErrInvalidInput
	}
	if err := s.requireExistingLocale(ctx, cmd.Locale); err != nil {
		return nil, err
	}
	article, err := s.repo.FindArticle(ctx, cmd.ArticleID)
	if err != nil {
		return nil, mapArticle(err)
	}
	translation, err := s.translation(ctx, cmd.ArticleID, cmd.Locale)
	if err != nil {
		return nil, err
	}
	categories, err := s.repo.ListArticleCategories(ctx, cmd.ArticleID)
	if err != nil {
		return nil, err
	}
	tags, err := s.repo.ListArticleTags(ctx, cmd.ArticleID, cmd.Locale)
	if err != nil {
		return nil, err
	}
	result := &ArticleDetailResult{ID: article.ID, AuthorUserID: article.AuthorUserID, Locale: translation.Locale, Title: translation.Title, Slug: translation.Slug, Summary: translation.Summary, Content: translation.Content, ContentFormat: translation.ContentFormat, Status: string(translation.Status), PublishedAt: translation.PublishedAt, SEOTitle: translation.SEOTitle, SEODescription: translation.SEODescription, CanonicalURL: translation.CanonicalURL, Categories: make([]ArticleCategoryResult, 0, len(categories)), Tags: make([]TagResult, 0, len(tags))}
	for _, category := range categories {
		result.Categories = append(result.Categories, ArticleCategoryResult{CategoryID: category.CategoryID, IsPrimary: category.IsPrimary})
	}
	for _, tag := range tags {
		result.Tags = append(result.Tags, *tagResult(tag.ID, &tag.TagTranslation))
	}
	return result, nil
}
