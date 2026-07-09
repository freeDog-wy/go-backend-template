package consumption

import (
	"context"
	"time"
)

// Repository 定义消费记录的持久化契约。
type Repository interface {
	Begin(ctx context.Context, command BeginCommand) (BeginResult, error)
	MarkDone(ctx context.Context, consumerGroup, messageKey string, processedAt time.Time) error
	MarkFailed(ctx context.Context, consumerGroup, messageKey, lastError string, failedAt time.Time) error
	MarkDead(ctx context.Context, consumerGroup, messageKey, lastError string, failedAt time.Time) error
}
