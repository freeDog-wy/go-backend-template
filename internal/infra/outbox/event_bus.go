package outbox

import (
	"context"
	"encoding/json"

	domainOutbox "github.com/freeDog-wy/go-backend-template/internal/domain/outbox"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"

	"go.opentelemetry.io/otel/trace"
)

// EventBus 是 shared.EventBus 的持久化实现：发布时先写本地 outbox，而不是直接打 MQ。
type EventBus struct {
	repo domainOutbox.Repository
}

func NewEventBus(repo domainOutbox.Repository) *EventBus {
	return &EventBus{repo: repo}
}

var _ shared.EventBus = (*EventBus)(nil)

// Publish 在业务事务里把领域事件序列化后写入 outbox 表。
func (b *EventBus) Publish(ctx context.Context, events ...shared.Event) error {
	if len(events) == 0 {
		return nil
	}

	traceID := extractTraceID(ctx)
	outboxEvents := make([]*domainOutbox.Event, 0, len(events))
	for _, event := range events {
		payload, err := json.Marshal(event)
		if err != nil {
			return err
		}

		outboxEvent, err := domainOutbox.NewEvent(event.EventName(), string(payload), traceID)
		if err != nil {
			return err
		}
		outboxEvents = append(outboxEvents, outboxEvent)
	}

	return b.repo.Create(ctx, outboxEvents...)
}

// extractTraceID 保留链路追踪信息，后续真正投递时继续透传给 worker。
func extractTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}
