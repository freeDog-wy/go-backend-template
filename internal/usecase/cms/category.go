package cms

import (
	"context"
	"errors"
	"strings"

	domainCMS "github.com/freeDog-wy/go-backend-template/internal/domain/cms"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
)

func (s *Service) CreateCategory(ctx context.Context, cmd CreateCategoryCmd) (*CategoryResult, error) {
	if err := validNameSlug(cmd.Name, cmd.Slug); err != nil {
		return nil, err
	}
	if err := s.requireLocale(ctx, cmd.Locale); err != nil {
		return nil, err
	}
	if cmd.ParentID != nil {
		if _, err := s.repo.FindCategory(ctx, *cmd.ParentID); err != nil {
			return nil, mapCategory(err)
		}
	}
	c := &domainCMS.Category{ParentID: cmd.ParentID, SortOrder: cmd.SortOrder, Enabled: true}
	tr := &domainCMS.CategoryTranslation{Locale: cmd.Locale, Name: strings.TrimSpace(cmd.Name), Slug: strings.TrimSpace(cmd.Slug), Description: cmd.Description, SEOTitle: cmd.SEOTitle, SEODescription: cmd.SEODescription}
	if err := s.repo.CreateCategory(ctx, c, tr); err != nil {
		return nil, err
	}
	return &CategoryResult{ID: c.ID, ParentID: c.ParentID, SortOrder: c.SortOrder, IsEnabled: c.Enabled, Locale: tr.Locale, Name: tr.Name, Slug: tr.Slug}, nil
}

// UpsertCategoryTranslation 更新分类翻译。已启用分类的 slug 变化会在同一事务中校验新
// 路径、保存翻译并为旧路径创建永久重定向，防止公开链接失效。
func (s *Service) UpsertCategoryTranslation(ctx context.Context, cmd UpsertCategoryTranslationCmd) (*CategoryResult, error) {
	if cmd.CategoryID == 0 {
		return nil, domainCMS.ErrInvalidInput
	}
	if err := validNameSlug(cmd.Name, cmd.Slug); err != nil {
		return nil, err
	}
	if err := s.requireExistingLocale(ctx, cmd.Locale); err != nil {
		return nil, err
	}
	category, err := s.repo.FindCategory(ctx, cmd.CategoryID)
	if err != nil {
		return nil, mapCategory(err)
	}
	locale := strings.TrimSpace(cmd.Locale)
	translation := &domainCMS.CategoryTranslation{CategoryID: cmd.CategoryID, Locale: locale, Name: strings.TrimSpace(cmd.Name), Slug: strings.TrimSpace(cmd.Slug), Description: cmd.Description, SEOTitle: cmd.SEOTitle, SEODescription: cmd.SEODescription}
	old, oldErr := s.repo.FindCategoryTranslation(ctx, cmd.CategoryID, locale)
	if oldErr != nil && !errors.Is(oldErr, shared.ErrNotFound) {
		return nil, oldErr
	}
	if err := s.tx.Do(ctx, func(ctx context.Context) error {
		if old != nil && old.Slug != translation.Slug {
			enabled, err := s.repo.LocaleEnabled(ctx, locale)
			if err != nil {
				return err
			}
			if category.Enabled && enabled {
				if err := s.ensureSlugAvailable(ctx, locale, categoryPath(locale, translation.Slug)); err != nil {
					return err
				}
			}
		}
		if err := s.repo.UpsertCategoryTranslation(ctx, translation); err != nil {
			return err
		}
		if old != nil && old.Slug != translation.Slug {
			enabled, err := s.repo.LocaleEnabled(ctx, locale)
			if err != nil {
				return err
			}
			if category.Enabled && enabled {
				redirect := &domainCMS.URLRedirect{Locale: locale, SourcePath: categoryPath(locale, old.Slug), TargetPath: categoryPath(locale, translation.Slug), StatusCode: 301}
				if err := s.repo.SaveURLRedirect(ctx, redirect); err != nil {
					return err
				}
				return s.publishAudit(ctx, cmd.ActorUserID, "category", cmd.CategoryID, auditActionSlugChanged, cmd.IP, cmd.UserAgent, map[string]any{"locale": locale, "old_slug": old.Slug, "new_slug": translation.Slug})
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return &CategoryResult{ID: category.ID, ParentID: category.ParentID, SortOrder: category.SortOrder, IsEnabled: category.Enabled, Locale: translation.Locale, Name: translation.Name, Slug: translation.Slug}, nil
}

func (s *Service) MoveCategory(ctx context.Context, cmd MoveCategoryCmd) error {
	if cmd.CategoryID == 0 {
		return domainCMS.ErrInvalidInput
	}
	if _, err := s.repo.FindCategory(ctx, cmd.CategoryID); err != nil {
		return mapCategory(err)
	}
	if cmd.ParentID != nil {
		if *cmd.ParentID == cmd.CategoryID {
			return domainCMS.ErrCategoryCycle
		}
		if _, err := s.repo.FindCategory(ctx, *cmd.ParentID); err != nil {
			return mapCategory(err)
		}
		descendant, err := s.repo.IsCategoryDescendant(ctx, cmd.CategoryID, *cmd.ParentID)
		if err != nil {
			return err
		}
		if descendant {
			return domainCMS.ErrCategoryCycle
		}
	}
	return s.tx.Do(ctx, func(ctx context.Context) error {
		if err := s.repo.MoveCategory(ctx, cmd.CategoryID, cmd.ParentID, cmd.SortOrder); err != nil {
			return err
		}
		return s.publishAudit(ctx, cmd.ActorUserID, "category", cmd.CategoryID, auditActionCategoryMoved, cmd.IP, cmd.UserAgent, map[string]any{"parent_id": cmd.ParentID, "sort_order": cmd.SortOrder})
	})
}
func (s *Service) UpdateCategory(ctx context.Context, cmd UpdateCategoryCmd) (*CategoryResult, error) {
	if cmd.CategoryID == 0 {
		return nil, domainCMS.ErrInvalidInput
	}
	category, err := s.repo.FindCategory(ctx, cmd.CategoryID)
	if err != nil {
		return nil, mapCategory(err)
	}
	if err := s.tx.Do(ctx, func(ctx context.Context) error {
		if err := s.repo.UpdateCategory(ctx, cmd.CategoryID, cmd.IsEnabled, cmd.SortOrder); err != nil {
			return err
		}
		return s.publishAudit(ctx, cmd.ActorUserID, "category", cmd.CategoryID, auditActionCategoryUpdated, cmd.IP, cmd.UserAgent, map[string]any{"old_enabled": category.Enabled, "new_enabled": cmd.IsEnabled, "old_sort_order": category.SortOrder, "new_sort_order": cmd.SortOrder})
	}); err != nil {
		return nil, err
	}
	category.Enabled, category.SortOrder = cmd.IsEnabled, cmd.SortOrder
	return &CategoryResult{ID: category.ID, ParentID: category.ParentID, SortOrder: category.SortOrder, IsEnabled: category.Enabled}, nil
}

// RenameCategory changes localized presentation fields while deliberately retaining
// the existing slug, so category URLs remain stable.
func (s *Service) RenameCategory(ctx context.Context, cmd RenameCategoryCmd) (*CategoryResult, error) {
	if cmd.CategoryID == 0 || strings.TrimSpace(cmd.Name) == "" {
		return nil, domainCMS.ErrInvalidInput
	}
	category, err := s.repo.FindCategory(ctx, cmd.CategoryID)
	if err != nil {
		return nil, mapCategory(err)
	}
	current, err := s.repo.FindCategoryTranslation(ctx, cmd.CategoryID, cmd.Locale)
	if err != nil {
		return nil, mapCategory(err)
	}
	translation := &domainCMS.CategoryTranslation{CategoryID: cmd.CategoryID, Locale: current.Locale, Name: strings.TrimSpace(cmd.Name), Slug: current.Slug, Description: current.Description, SEOTitle: current.SEOTitle, SEODescription: current.SEODescription}
	if err := validNameSlug(translation.Name, translation.Slug); err != nil {
		return nil, err
	}
	if err := s.tx.Do(ctx, func(ctx context.Context) error {
		if err := s.repo.UpsertCategoryTranslation(ctx, translation); err != nil {
			return err
		}
		return s.publishAudit(ctx, cmd.ActorUserID, "category", cmd.CategoryID, auditActionCategoryRenamed, cmd.IP, cmd.UserAgent, map[string]any{"locale": translation.Locale, "old_name": current.Name, "new_name": translation.Name, "slug": translation.Slug})
	}); err != nil {
		return nil, err
	}
	return &CategoryResult{ID: category.ID, ParentID: category.ParentID, SortOrder: category.SortOrder, IsEnabled: category.Enabled, Locale: translation.Locale, Name: translation.Name, Slug: translation.Slug}, nil
}

func (s *Service) DeleteCategory(ctx context.Context, cmd DeleteCategoryCmd) error {
	if cmd.CategoryID == 0 {
		return domainCMS.ErrInvalidInput
	}
	if _, err := s.repo.FindCategory(ctx, cmd.CategoryID); err != nil {
		return mapCategory(err)
	}
	return s.tx.Do(ctx, func(ctx context.Context) error {
		articleCount, err := s.repo.CountCategoryArticleReferences(ctx, cmd.CategoryID)
		if err != nil {
			return err
		}
		if articleCount > 0 {
			return domainCMS.ErrCategoryInUse
		}
		childCount, err := s.repo.CountCategoryChildren(ctx, cmd.CategoryID)
		if err != nil {
			return err
		}
		if childCount > 0 {
			return domainCMS.ErrCategoryHasChildren
		}
		if err := s.repo.DeleteCategory(ctx, cmd.CategoryID); err != nil {
			return mapCategory(err)
		}
		return s.publishAudit(ctx, cmd.ActorUserID, "category", cmd.CategoryID, auditActionCategoryDeleted, cmd.IP, cmd.UserAgent, nil)
	})
}

func (s *Service) ListCategories(ctx context.Context, cmd ListCategoriesCmd) ([]*CategoryTreeResult, error) {
	if err := s.requireLocale(ctx, cmd.Locale); err != nil {
		return nil, err
	}
	items, err := s.repo.ListCategoryTreeItems(ctx, cmd.Locale)
	if err != nil {
		return nil, err
	}
	return categoryTree(items), nil
}
func (s *Service) ListPublishedCategories(ctx context.Context, locale string) ([]*CategoryTreeResult, error) {
	if err := s.requireLocale(ctx, locale); err != nil {
		return nil, err
	}
	items, err := s.repo.ListPublicCategoryTreeItems(ctx, locale)
	if err != nil {
		return nil, err
	}
	return categoryTree(items), nil
}
func categoryTree(items []*domainCMS.CategoryTreeItem) []*CategoryTreeResult {
	byID := make(map[uint]*CategoryTreeResult, len(items))
	for _, item := range items {
		byID[item.ID] = &CategoryTreeResult{ID: item.ID, ParentID: item.ParentID, SortOrder: item.SortOrder, IsEnabled: item.Enabled, Name: item.Name, Slug: item.Slug, Description: item.Description, Children: make([]*CategoryTreeResult, 0)}
	}
	roots := make([]*CategoryTreeResult, 0)
	for _, item := range items {
		node := byID[item.ID]
		if item.ParentID != nil {
			if parent, ok := byID[*item.ParentID]; ok {
				parent.Children = append(parent.Children, node)
				continue
			}
		}
		roots = append(roots, node)
	}
	return roots
}
