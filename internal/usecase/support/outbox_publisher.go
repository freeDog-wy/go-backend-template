package support

import (
	"context"
	"strconv"
	"time"

	domainOutbox "github.com/freeDog-wy/go-backend-template/internal/domain/outbox"
	"github.com/freeDog-wy/go-backend-template/pkg/logger"
)

// OutboxPublisher 负责扫描本地 outbox 并把事件真正投递到外部消息系统。
type OutboxPublisher struct {
	repo      domainOutbox.Repository
	publisher domainOutbox.Publisher
	logger    logger.Logger
	batchSize int
}

func NewOutboxPublisher(
	repo domainOutbox.Repository,
	publisher domainOutbox.Publisher,
	logger logger.Logger,
	batchSize int,
) *OutboxPublisher {
	if batchSize <= 0 {
		batchSize = 100
	}

	return &OutboxPublisher{
		repo:      repo,
		publisher: publisher,
		logger:    logger,
		batchSize: batchSize,
	}
}

// PublishPending 每次抓取一批未发布事件，按顺序投递，成功后再回写 published 状态。
func (p *OutboxPublisher) PublishPending(ctx context.Context) error {
	events, err := p.repo.ListUnpublished(ctx, p.batchSize)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	publishedIDs := make([]uint, 0, len(events))
	var publishErr error
	for _, event := range events {
		if err := p.publisher.Publish(
			ctx,
			uintString(event.GetID()),
			event.GetEventName(),
			[]byte(event.GetPayload()),
			event.GetTraceID(),
		); err != nil {
			publishErr = err
			if p.logger != nil {
				p.logger.Error("outbox publish failed", "event", event.GetEventName(), "outbox_id", event.GetID(), "error", err)
			}
			break
		}
		publishedIDs = append(publishedIDs, event.GetID())
	}

	if err := p.repo.MarkPublished(ctx, publishedIDs, time.Now()); err != nil {
		return err
	}

	if p.logger != nil && len(publishedIDs) > 0 {
		p.logger.Info("outbox events published", "count", len(publishedIDs))
	}

	return publishErr
}

func uintString(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}
