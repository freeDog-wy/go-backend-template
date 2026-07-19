package cms

import (
	"context"
	"time"

	domainCMS "github.com/freeDog-wy/go-backend-template/internal/domain/cms"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
	modelCMS "github.com/freeDog-wy/go-backend-template/internal/model/cms"
	"gorm.io/gorm/clause"
)

func (r *Repository) CreateTag(ctx context.Context, tag *domainCMS.Tag, translation *domainCMS.TagTranslation) error {
	m := modelCMS.Tag{}
	if err := r.conn(ctx).Create(&m).Error; err != nil {
		return err
	}
	tag.ID, tag.CreatedAt, tag.UpdatedAt = m.ID, m.CreatedAt, m.UpdatedAt
	return r.conn(ctx).Create(&modelCMS.TagTranslation{TagID: m.ID, Locale: translation.Locale, Name: translation.Name, Slug: translation.Slug}).Error
}

func (r *Repository) FindTag(ctx context.Context, id uint) (*domainCMS.Tag, error) {
	var m modelCMS.Tag
	if err := r.conn(ctx).First(&m, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &domainCMS.Tag{ID: m.ID, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}, nil
}

func (r *Repository) FindTagTranslation(ctx context.Context, tagID uint, locale string) (*domainCMS.TagTranslation, error) {
	var m modelCMS.TagTranslation
	if err := r.conn(ctx).Where("tag_id = ? AND locale = ?", tagID, locale).First(&m).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &domainCMS.TagTranslation{TagID: m.TagID, Locale: m.Locale, Name: m.Name, Slug: m.Slug}, nil
}

func (r *Repository) UpsertTagTranslation(ctx context.Context, translation *domainCMS.TagTranslation) error {
	m := modelCMS.TagTranslation{TagID: translation.TagID, Locale: translation.Locale, Name: translation.Name, Slug: translation.Slug}
	return r.conn(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "tag_id"}, {Name: "locale"}}, DoUpdates: clause.Assignments(map[string]any{"name": m.Name, "slug": m.Slug, "updated_at": time.Now()})}).Create(&m).Error
}

func (r *Repository) ListTags(ctx context.Context, locale string, page shared.PageQuery) ([]*domainCMS.TagListItem, int64, error) {
	db := r.conn(ctx).Table("tag_translations").Joins("JOIN tags ON tags.id = tag_translations.tag_id").Where("tag_translations.locale = ?", locale)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	type row struct {
		TagID                uint
		Name, Slug           string
		CreatedAt, UpdatedAt time.Time
	}
	var rows []row
	if err := db.Select("tags.id AS tag_id, tags.created_at, tags.updated_at, tag_translations.name, tag_translations.slug").Order("tag_translations.name, tags.id").Limit(page.PerPage).Offset(page.Offset()).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	result := make([]*domainCMS.TagListItem, 0, len(rows))
	for _, row := range rows {
		result = append(result, &domainCMS.TagListItem{Tag: domainCMS.Tag{ID: row.TagID, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, TagTranslation: domainCMS.TagTranslation{TagID: row.TagID, Locale: locale, Name: row.Name, Slug: row.Slug}})
	}
	return result, total, nil
}
