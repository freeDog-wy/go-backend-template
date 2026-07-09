package outbox

import (
	"context"
	"time"
)

// Repository 定义 outbox 本地消息表的持久化契约。
type Repository interface {
	Create(ctx context.Context, events ...*Event) error
	ListUnpublished(ctx context.Context, limit int) ([]*Event, error)
	MarkPublished(ctx context.Context, ids []uint, publishedAt time.Time) error
}

// Publisher 定义把原始事件投递到外部消息系统的能力。
type Publisher interface {
	Publish(ctx context.Context, messageKey, eventName string, payload []byte, traceID string) error
}
