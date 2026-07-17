package audit

import (
	"context"

	"github.com/freeDog-wy/go-backend-template/pkg/logger"
	"go.opentelemetry.io/otel/trace"
)

type Consumer struct {
	repo   Writer
	logger logger.Logger
}

func NewConsumer(repo Writer, logger logger.Logger) *Consumer {
	return &Consumer{
		repo:   repo,
		logger: logger,
	}
}

func (c *Consumer) OnLogRequested(ctx context.Context, evt LogRequested) error {
	log, err := NewAuditLog(
		evt.ActorUserID,
		evt.TargetType,
		evt.TargetID,
		evt.Action,
		evt.Result,
		evt.IP,
		evt.UserAgent,
		traceIDFromContext(ctx),
		evt.Metadata,
	)
	if err != nil {
		return err
	}

	if err := c.repo.Create(ctx, log); err != nil {
		if c.logger != nil {
			c.logger.Error("audit log persist failed", "action", evt.Action, "error", err)
		}
		return err
	}
	return nil
}

func traceIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}
