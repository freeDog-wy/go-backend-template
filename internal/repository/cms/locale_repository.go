package cms

import (
	"context"

	domainCMS "github.com/freeDog-wy/go-backend-template/internal/domain/cms"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
	modelCMS "github.com/freeDog-wy/go-backend-template/internal/model/cms"
)

func (r *Repository) LocaleEnabled(ctx context.Context, code string) (bool, error) {
	var count int64
	err := r.conn(ctx).Table("locales").Where("code = ? AND is_enabled", code).Count(&count).Error
	return count == 1, err
}

func (r *Repository) ListLocales(ctx context.Context) ([]*domainCMS.Locale, error) {
	var models []modelCMS.Locale
	if err := r.conn(ctx).Order("sort_order, code").Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*domainCMS.Locale, 0, len(models))
	for _, m := range models {
		result = append(result, localeEntity(m))
	}
	return result, nil
}

func (r *Repository) FindLocale(ctx context.Context, code string) (*domainCMS.Locale, error) {
	var m modelCMS.Locale
	if err := r.conn(ctx).First(&m, "code = ?", code).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return localeEntity(m), nil
}

func (r *Repository) CreateLocale(ctx context.Context, locale *domainCMS.Locale) error {
	m := localeModel(locale)
	if err := r.conn(ctx).Create(&m).Error; err != nil {
		return err
	}
	locale.CreatedAt, locale.UpdatedAt = m.CreatedAt, m.UpdatedAt
	return nil
}

func (r *Repository) UpdateLocale(ctx context.Context, locale *domainCMS.Locale) error {
	m := localeModel(locale)
	result := r.conn(ctx).Model(&modelCMS.Locale{}).Where("code = ?", m.Code).Updates(map[string]any{"name": m.Name, "is_enabled": m.IsEnabled, "sort_order": m.SortOrder})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

func (r *Repository) SetDefaultLocale(ctx context.Context, code string) error {
	db := r.conn(ctx)
	if err := db.Model(&modelCMS.Locale{}).Where("is_default").Update("is_default", false).Error; err != nil {
		return err
	}
	result := db.Model(&modelCMS.Locale{}).Where("code = ? AND is_enabled", code).Update("is_default", true)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

func (r *Repository) CountEnabledLocales(ctx context.Context) (int64, error) {
	var count int64
	err := r.conn(ctx).Model(&modelCMS.Locale{}).Where("is_enabled").Count(&count).Error
	return count, err
}

func localeModel(locale *domainCMS.Locale) modelCMS.Locale {
	return modelCMS.Locale{Code: locale.Code, Name: locale.Name, IsDefault: locale.IsDefault, IsEnabled: locale.IsEnabled, SortOrder: locale.SortOrder}
}

func localeEntity(model modelCMS.Locale) *domainCMS.Locale {
	return &domainCMS.Locale{Code: model.Code, Name: model.Name, IsDefault: model.IsDefault, IsEnabled: model.IsEnabled, SortOrder: model.SortOrder, CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt}
}
