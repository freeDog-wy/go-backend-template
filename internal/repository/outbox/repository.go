package outbox

import (
	"context"
	"time"

	domainOutbox "github.com/freeDog-wy/go-backend-template/internal/domain/outbox"
	"github.com/freeDog-wy/go-backend-template/internal/infra/database"
	modelOutbox "github.com/freeDog-wy/go-backend-template/internal/model/outbox"

	"gorm.io/gorm"
)

// Repository 基于 GORM 实现 outbox 本地消息表读写。
type Repository struct {
	db *gorm.DB
}

var _ domainOutbox.Repository = (*Repository)(nil)

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create 在当前事务上下文内批量写入待发布事件。
func (r *Repository) Create(ctx context.Context, events ...*domainOutbox.Event) error {
	if len(events) == 0 {
		return nil
	}

	models := make([]*modelOutbox.Event, 0, len(events))
	for _, event := range events {
		models = append(models, modelOutbox.FromEntity(event))
	}

	return database.DB(ctx, r.db).Create(&models).Error
}

// ListUnpublished 按主键顺序抓取一批尚未投递的事件，供 cron publisher 扫描。
func (r *Repository) ListUnpublished(ctx context.Context, limit int) ([]*domainOutbox.Event, error) {
	if limit <= 0 {
		limit = 100
	}

	var models []*modelOutbox.Event
	if err := database.DB(ctx, r.db).
		Where("published_at IS NULL").
		Order("id ASC").
		Limit(limit).
		Find(&models).Error; err != nil {
		return nil, err
	}

	events := make([]*domainOutbox.Event, 0, len(models))
	for _, model := range models {
		events = append(events, model.ToEntity())
	}
	return events, nil
}

// MarkPublished 只更新仍未发布的记录，避免重复覆盖状态。
func (r *Repository) MarkPublished(ctx context.Context, ids []uint, publishedAt time.Time) error {
	if len(ids) == 0 {
		return nil
	}

	return database.DB(ctx, r.db).
		Model(&modelOutbox.Event{}).
		Where("id IN ? AND published_at IS NULL", ids).
		Update("published_at", publishedAt).Error
}
