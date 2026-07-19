package cms

import (
	"context"
	"strings"
	
	domainCMS "github.com/freeDog-wy/go-backend-template/internal/domain/cms"
)

func (s *Service) ListLocales(ctx context.Context) ([]*LocaleResult, error) {
	locales, err := s.repo.ListLocales(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*LocaleResult, 0, len(locales))
	for _, locale := range locales {
		result = append(result, localeResult(locale))
	}
	return result, nil
}
func (s *Service) ListPublishedLocales(ctx context.Context) ([]*LocaleResult, error) {
	locales, err := s.repo.ListLocales(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*LocaleResult, 0, len(locales))
	for _, locale := range locales {
		if locale.IsEnabled {
			result = append(result, localeResult(locale))
		}
	}
	return result, nil
}

func (s *Service) CreateLocale(ctx context.Context, cmd CreateLocaleCmd) (*LocaleResult, error) {
	code, name := strings.TrimSpace(cmd.Code), strings.TrimSpace(cmd.Name)
	if !validLocale(code) || name == "" {
		return nil, domainCMS.ErrInvalidInput
	}
	locale := &domainCMS.Locale{Code: code, Name: name, IsEnabled: cmd.IsEnabled, SortOrder: cmd.SortOrder}
	if err := s.tx.Do(ctx, func(ctx context.Context) error {
		if err := s.repo.CreateLocale(ctx, locale); err != nil {
			return err
		}
		return s.publishAuditText(ctx, cmd.ActorUserID, "locale", locale.Code, auditActionLocaleCreated, cmd.IP, cmd.UserAgent, map[string]any{"name": locale.Name, "sort_order": locale.SortOrder})
	}); err != nil {
		return nil, err
	}
	return localeResult(locale), nil
}
func (s *Service) UpdateLocale(ctx context.Context, cmd UpdateLocaleCmd) (*LocaleResult, error) {
	if !validLocale(strings.TrimSpace(cmd.Code)) || strings.TrimSpace(cmd.Name) == "" {
		return nil, domainCMS.ErrInvalidInput
	}
	locale, err := s.repo.FindLocale(ctx, cmd.Code)
	if err != nil {
		return nil, mapLocale(err)
	}
	if locale.IsDefault && !cmd.IsEnabled {
		return nil, domainCMS.ErrLocaleDefault
	}
	if locale.IsEnabled && !cmd.IsEnabled {
		count, err := s.repo.CountEnabledLocales(ctx)
		if err != nil {
			return nil, err
		}
		if count <= 1 {
			return nil, domainCMS.ErrLastEnabledLocale
		}
	}
	if cmd.IsDefault && !cmd.IsEnabled {
		return nil, domainCMS.ErrInvalidInput
	}
	oldName, oldEnabled, oldSortOrder, oldDefault := locale.Name, locale.IsEnabled, locale.SortOrder, locale.IsDefault
	locale.Name, locale.IsEnabled, locale.SortOrder = strings.TrimSpace(cmd.Name), cmd.IsEnabled, cmd.SortOrder
	if err := s.tx.Do(ctx, func(ctx context.Context) error {
		if err := s.repo.UpdateLocale(ctx, locale); err != nil {
			return err
		}
		if cmd.IsDefault && !locale.IsDefault {
			if err := s.repo.SetDefaultLocale(ctx, locale.Code); err != nil {
				return err
			}
			locale.IsDefault = true
		}
		return s.publishAuditText(ctx, cmd.ActorUserID, "locale", locale.Code, auditActionLocaleUpdated, cmd.IP, cmd.UserAgent, map[string]any{"old_name": oldName, "new_name": locale.Name, "old_enabled": oldEnabled, "new_enabled": locale.IsEnabled, "old_sort_order": oldSortOrder, "new_sort_order": locale.SortOrder, "old_default": oldDefault, "new_default": locale.IsDefault})
	}); err != nil {
		return nil, err
	}
	return localeResult(locale), nil
}

