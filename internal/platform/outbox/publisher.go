package outbox

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/freeDog-wy/go-backend-template/pkg/logger"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var outboxPublisherTracer = otel.Tracer("github.com/freeDog-wy/go-backend-template/internal/platform/outbox")

type Publisher interface {
	Publish(ctx context.Context, messageKey, eventName string, payload []byte, traceID, traceContext string) error
}

type OutboxPublisher struct {
	repo      *Repository
	publisher Publisher
	logger    logger.Logger
	batchSize int
	claimTTL  time.Duration
}

func NewOutboxPublisher(
	repo *Repository,
	publisher Publisher,
	logger logger.Logger,
	batchSize int,
	claimTTL time.Duration,
) *OutboxPublisher {
	if batchSize <= 0 {
		batchSize = 100
	}
	if claimTTL <= 0 {
		claimTTL = time.Minute
	}

	return &OutboxPublisher{
		repo:      repo,
		publisher: publisher,
		logger:    logger,
		batchSize: batchSize,
		claimTTL:  claimTTL,
	}
}

func (p *OutboxPublisher) PublishPending(ctx context.Context) (err error) {
	ctx, span := outboxPublisherTracer.Start(ctx, "outbox.publish_pending")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}
		span.End()
	}()

	span.SetAttributes(attribute.Int("outbox.batch_size", p.batchSize))

	claimant := uuid.NewString()
	events, err := p.repo.ClaimUnpublished(ctx, claimant, time.Now(), p.claimTTL, p.batchSize)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	published := 0
	for index, event := range events {
		if err := p.publisher.Publish(
			ctx,
			uintString(event.GetID()),
			event.GetEventName(),
			[]byte(event.GetPayload()),
			event.GetTraceID(),
			event.GetTraceContext(),
		); err != nil {
			if releaseErr := p.repo.ReleaseClaims(ctx, eventIDs(events[index:]), claimant); releaseErr != nil && p.logger != nil {
				p.logger.Error("release outbox claims failed", "error", releaseErr)
			}
			if p.logger != nil {
				p.logger.Error("outbox publish failed", "event", event.GetEventName(), "outbox_id", event.GetID(), "error", err)
			}
			return err
		}

		marked, err := p.repo.MarkPublished(ctx, event.GetID(), claimant, time.Now())
		if err != nil {
			return err
		}
		if !marked {
			return fmt.Errorf("outbox claim lost before marking event %d as published", event.GetID())
		}
		published++
	}

	span.SetAttributes(
		attribute.Int("outbox.fetched", len(events)),
		attribute.Int("outbox.published", published),
	)
	if p.logger != nil && published > 0 {
		p.logger.Info("outbox events published", "count", published)
	}

	return nil
}

func eventIDs(events []*Event) []uint {
	ids := make([]uint, 0, len(events))
	for _, event := range events {
		ids = append(ids, event.GetID())
	}
	return ids
}

func uintString(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}
