package mq

import (
	"context"

	domainOutbox "github.com/freeDog-wy/go-backend-template/internal/domain/outbox"
)

// OutboxPublisherAdapter 把 outbox 的领域发布契约适配到 mq.Publisher。
type OutboxPublisherAdapter struct {
	publisher Publisher
}

func NewOutboxPublisherAdapter(publisher Publisher) *OutboxPublisherAdapter {
	return &OutboxPublisherAdapter{publisher: publisher}
}

var _ domainOutbox.Publisher = (*OutboxPublisherAdapter)(nil)

func (a *OutboxPublisherAdapter) Publish(ctx context.Context, messageKey, eventName string, payload []byte, traceID string) error {
	return a.publisher.Publish(ctx, Message{
		Key:     messageKey,
		Event:   eventName,
		Payload: payload,
		TraceID: traceID,
	})
}
