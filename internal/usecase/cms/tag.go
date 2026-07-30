package cms

import (
	"context"
	"errors"
	"strings"

	domainCMS "github.com/freeDog-wy/go-backend-template/internal/domain/cms"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
)

func (s *Service) CreateTag(ctx context.Context, cmd CreateTagCmd) (*TagResult, error) {
	if err := validNameSlug(cmd.Name, cmd.Slug); err != nil {
		return nil, err
	}
	if err := s.requireLocale(ctx, cmd.Locale); err != nil {
		return nil, err
	}
	tag := &domainCMS.Tag{Enabled: true}
	tr := &domainCMS.TagTranslation{Locale: strings.TrimSpace(cmd.Locale), Name: strings.TrimSpace(cmd.Name), Slug: strings.TrimSpace(cmd.Slug)}
	if err := s.tx.Do(ctx, func(ctx context.Context) error {
		if err := s.ensureSlugAvailable(ctx, tr.Locale, tagPath(tr.Locale, tr.Slug)); err != nil {
			return err
		}
		if err := s.repo.CreateTag(ctx, tag, tr); err != nil {
			return err
		}
		return s.publishAudit(ctx, cmd.ActorUserID, "tag", tag.ID, auditActionTagCreated, cmd.IP, cmd.UserAgent, map[string]any{"locale": tr.Locale, "slug": tr.Slug})
	}); err != nil {
		return nil, err
	}
	return tagResult(tag, tr), nil
}
func (s *Service) ListTags(ctx context.Context, cmd ListTagsCmd) ([]*TagResult, shared.PageResult, error) {
	if err := s.requireExistingLocale(ctx, cmd.Locale); err != nil {
		return nil, shared.PageResult{}, err
	}
	page := shared.NewPageQuery(cmd.Page.Page, cmd.Page.PerPage)
	items, total, err := s.repo.ListTags(ctx, cmd.Locale, page)
	if err != nil {
		return nil, shared.PageResult{}, err
	}
	out := make([]*TagResult, 0, len(items))
	for _, v := range items {
		out = append(out, tagResult(&v.Tag, &v.TagTranslation))
	}
	return out, shared.PageResult{Page: page.Page, PerPage: page.PerPage, Total: total}, nil
}

func (s *Service) UpsertTagTranslation(ctx context.Context, cmd UpsertTagTranslationCmd) (*TagResult, error) {
	if cmd.TagID == 0 {
		return nil, domainCMS.ErrInvalidInput
	}
	if err := validNameSlug(cmd.Name, cmd.Slug); err != nil {
		return nil, err
	}
	if err := s.requireExistingLocale(ctx, cmd.Locale); err != nil {
		return nil, err
	}
	tag, err := s.repo.FindTag(ctx, cmd.TagID)
	if err != nil {
		return nil, mapTag(err)
	}
	tr := &domainCMS.TagTranslation{TagID: cmd.TagID, Locale: strings.TrimSpace(cmd.Locale), Name: strings.TrimSpace(cmd.Name), Slug: strings.TrimSpace(cmd.Slug)}
	old, err := s.repo.FindTagTranslation(ctx, cmd.TagID, tr.Locale)
	if err != nil && !errors.Is(err, shared.ErrNotFound) {
		return nil, err
	}
	if err := s.tx.Do(ctx, func(ctx context.Context) error {
		if old != nil && old.Slug != tr.Slug {
			enabled, e := s.repo.LocaleEnabled(ctx, tr.Locale)
			if e != nil {
				return e
			}
			if enabled {
				if e = s.ensureSlugAvailable(ctx, tr.Locale, tagPath(tr.Locale, tr.Slug)); e != nil {
					return e
				}
			}
		}
		if e := s.repo.UpsertTagTranslation(ctx, tr); e != nil {
			return e
		}
		if old != nil && old.Slug != tr.Slug {
			enabled, e := s.repo.LocaleEnabled(ctx, tr.Locale)
			if e != nil {
				return e
			}
			if enabled {
				return s.repo.SaveURLRedirect(ctx, &domainCMS.URLRedirect{Locale: tr.Locale, SourcePath: tagPath(tr.Locale, old.Slug), TargetPath: tagPath(tr.Locale, tr.Slug), StatusCode: 301})
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return tagResult(tag, tr), nil
}

func (s *Service) UpdateTag(ctx context.Context, cmd UpdateTagCmd) (*TagResult, error) {
	if cmd.TagID == 0 {
		return nil, domainCMS.ErrInvalidInput
	}
	tag, err := s.repo.FindTag(ctx, cmd.TagID)
	if err != nil {
		return nil, mapTag(err)
	}
	if err := s.tx.Do(ctx, func(ctx context.Context) error {
		if err := s.repo.UpdateTag(ctx, cmd.TagID, cmd.IsEnabled); err != nil {
			return err
		}
		return s.publishAudit(ctx, cmd.ActorUserID, "tag", cmd.TagID, auditActionTagUpdated, cmd.IP, cmd.UserAgent, map[string]any{"old_enabled": tag.Enabled, "new_enabled": cmd.IsEnabled})
	}); err != nil {
		return nil, err
	}
	tag.Enabled = cmd.IsEnabled
	return &TagResult{ID: tag.ID, IsEnabled: tag.Enabled}, nil
}

// RenameTag changes the localized label while retaining its slug and public URL.
func (s *Service) RenameTag(ctx context.Context, cmd RenameTagCmd) (*TagResult, error) {
	if cmd.TagID == 0 || strings.TrimSpace(cmd.Name) == "" {
		return nil, domainCMS.ErrInvalidInput
	}
	tag, err := s.repo.FindTag(ctx, cmd.TagID)
	if err != nil {
		return nil, mapTag(err)
	}
	current, err := s.repo.FindTagTranslation(ctx, cmd.TagID, cmd.Locale)
	if err != nil {
		return nil, mapTag(err)
	}
	translation := &domainCMS.TagTranslation{TagID: cmd.TagID, Locale: current.Locale, Name: strings.TrimSpace(cmd.Name), Slug: current.Slug}
	if err := validNameSlug(translation.Name, translation.Slug); err != nil {
		return nil, err
	}
	if err := s.tx.Do(ctx, func(ctx context.Context) error {
		if err := s.repo.UpsertTagTranslation(ctx, translation); err != nil {
			return err
		}
		return s.publishAudit(ctx, cmd.ActorUserID, "tag", cmd.TagID, auditActionTagRenamed, cmd.IP, cmd.UserAgent, map[string]any{"locale": translation.Locale, "old_name": current.Name, "new_name": translation.Name, "slug": translation.Slug})
	}); err != nil {
		return nil, err
	}
	return tagResult(tag, translation), nil
}

func (s *Service) ListPublishedTags(ctx context.Context, cmd ListPublicTagsCmd) ([]*TagResult, shared.PageResult, error) {
	if err := s.requireLocale(ctx, cmd.Locale); err != nil {
		return nil, shared.PageResult{}, err
	}
	page := shared.NewPageQuery(cmd.Page.Page, cmd.Page.PerPage)
	items, total, err := s.repo.ListPublicTags(ctx, cmd.Locale, page)
	if err != nil {
		return nil, shared.PageResult{}, err
	}
	result := make([]*TagResult, 0, len(items))
	for _, item := range items {
		result = append(result, tagResult(&item.Tag, &item.TagTranslation))
	}
	return result, shared.PageResult{Page: page.Page, PerPage: page.PerPage, Total: total}, nil
}
