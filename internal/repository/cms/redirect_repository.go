package cms

import (
	"context"

	domainCMS "github.com/freeDog-wy/go-backend-template/internal/domain/cms"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
	modelCMS "github.com/freeDog-wy/go-backend-template/internal/model/cms"
	"gorm.io/gorm/clause"
)

func (r *Repository) RedirectSourceExists(ctx context.Context, locale, sourcePath string) (bool, error) {
	var count int64
	err := r.conn(ctx).Model(&modelCMS.URLRedirect{}).Where("locale = ? AND source_path = ?", locale, sourcePath).Count(&count).Error
	return count > 0, err
}

func (r *Repository) SaveURLRedirect(ctx context.Context, redirect *domainCMS.URLRedirect) error {
	db := r.conn(ctx)
	if err := db.Model(&modelCMS.URLRedirect{}).Where("locale = ? AND target_path = ?", redirect.Locale, redirect.SourcePath).Update("target_path", redirect.TargetPath).Error; err != nil {
		return err
	}
	m := modelCMS.URLRedirect{Locale: redirect.Locale, SourcePath: redirect.SourcePath, TargetPath: redirect.TargetPath, StatusCode: redirect.StatusCode}
	if m.StatusCode == 0 {
		m.StatusCode = 301
	}
	return db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "locale"}, {Name: "source_path"}}, DoUpdates: clause.Assignments(map[string]any{"target_path": m.TargetPath, "status_code": m.StatusCode})}).Create(&m).Error
}

func (r *Repository) FindURLRedirect(ctx context.Context, locale, sourcePath string) (*domainCMS.URLRedirect, error) {
	var m modelCMS.URLRedirect
	if err := r.conn(ctx).Where("locale = ? AND source_path = ?", locale, sourcePath).First(&m).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &domainCMS.URLRedirect{Locale: m.Locale, SourcePath: m.SourcePath, TargetPath: m.TargetPath, StatusCode: m.StatusCode, CreatedAt: m.CreatedAt}, nil
}

func (r *Repository) ListURLRedirects(ctx context.Context, locale string, page shared.PageQuery) ([]domainCMS.URLRedirect, int64, error) {
	db := r.conn(ctx).Model(&modelCMS.URLRedirect{}).Where("locale = ?", locale)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []modelCMS.URLRedirect
	if err := db.Order("source_path, id").Limit(page.PerPage).Offset(page.Offset()).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	result := make([]domainCMS.URLRedirect, 0, len(models))
	for _, m := range models {
		result = append(result, domainCMS.URLRedirect{Locale: m.Locale, SourcePath: m.SourcePath, TargetPath: m.TargetPath, StatusCode: m.StatusCode, CreatedAt: m.CreatedAt})
	}
	return result, total, nil
}
