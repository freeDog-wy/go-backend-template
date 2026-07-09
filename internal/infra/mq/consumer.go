package mq

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/freeDog-wy/go-backend-template/pkg/logger"

	"github.com/redis/go-redis/v9"
)

const defaultPendingScanStart = "0-0"

type ctxKey struct{}

func TraceIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKey{}).(string); ok {
		return v
	}
	return ""
}

type ConsumerConfig struct {
	ReadCount           int64
	ReadBlock           time.Duration
	PendingMinIdle      time.Duration
	PendingReclaimBatch int64
	MaxRetries          int64
	IdempotencyTTL      time.Duration
	ProcessingLockTTL   time.Duration
	DeadLetterStream    string
}

type RedisConsumer struct {
	rdb      *redis.Client
	stream   string
	group    string
	consumer string
	handlers map[string]EventHandler
	logger   logger.Logger
	config   ConsumerConfig
}

var _ Consumer = (*RedisConsumer)(nil)

func NewRedisConsumer(rdb *redis.Client, stream, group, consumer string, log logger.Logger, cfg ConsumerConfig) *RedisConsumer {
	if cfg.ReadCount <= 0 {
		cfg.ReadCount = 10
	}
	if cfg.ReadBlock <= 0 {
		cfg.ReadBlock = 1 * time.Second
	}
	if cfg.PendingMinIdle <= 0 {
		cfg.PendingMinIdle = 30 * time.Second
	}
	if cfg.PendingReclaimBatch <= 0 {
		cfg.PendingReclaimBatch = 10
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 10
	}
	if cfg.IdempotencyTTL <= 0 {
		cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	if cfg.ProcessingLockTTL <= 0 {
		cfg.ProcessingLockTTL = 5 * time.Minute
	}
	if strings.TrimSpace(cfg.DeadLetterStream) == "" {
		cfg.DeadLetterStream = stream + ".dlq"
	}

	return &RedisConsumer{
		rdb:      rdb,
		stream:   stream,
		group:    group,
		consumer: consumer,
		handlers: make(map[string]EventHandler),
		logger:   log,
		config:   cfg,
	}
}

func (c *RedisConsumer) Handle(eventName string, fn EventHandler) {
	c.handlers[eventName] = fn
}

func (c *RedisConsumer) Run(ctx context.Context) error {
	if err := c.createGroup(ctx); err != nil {
		return err
	}

	c.logger.Info("consumer started", "group", c.group, "consumer", c.consumer, "stream", c.stream)

	reclaimStart := defaultPendingScanStart
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		reclaimed, nextStart, err := c.reclaimPending(ctx, reclaimStart)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				continue
			}
			c.logger.Error("xautoclaim error", "error", err)
			time.Sleep(time.Second)
			continue
		}
		reclaimStart = nextStart
		if len(reclaimed) > 0 {
			c.processMessages(ctx, reclaimed)
			continue
		}

		msgs, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.group,
			Consumer: c.consumer,
			Streams:  []string{c.stream, ">"},
			Count:    c.config.ReadCount,
			Block:    c.config.ReadBlock,
		}).Result()

		if err != nil {
			if errors.Is(err, redis.Nil) || errors.Is(err, context.Canceled) {
				continue
			}
			c.logger.Error("xreadgroup error", "error", err)
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range msgs {
			c.processMessages(ctx, stream.Messages)
		}
	}
}

func (c *RedisConsumer) createGroup(ctx context.Context) error {
	err := c.rdb.XGroupCreateMkStream(ctx, c.stream, c.group, "$").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

func (c *RedisConsumer) reclaimPending(ctx context.Context, start string) ([]redis.XMessage, string, error) {
	msgs, nextStart, err := c.rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   c.stream,
		Group:    c.group,
		Consumer: c.consumer,
		MinIdle:  c.config.PendingMinIdle,
		Start:    start,
		Count:    c.config.PendingReclaimBatch,
	}).Result()
	if err != nil {
		return nil, start, err
	}

	if nextStart == "" || nextStart == "0-0" {
		nextStart = defaultPendingScanStart
	}

	return msgs, nextStart, nil
}

func (c *RedisConsumer) processMessages(ctx context.Context, messages []redis.XMessage) {
	for _, msg := range messages {
		c.dispatch(ctx, msg)
	}
}

func (c *RedisConsumer) dispatch(ctx context.Context, msg redis.XMessage) {
	eventMessage := c.decodeMessage(msg)
	messageKey := c.messageKey(msg, eventMessage)

	state, err := c.idempotencyState(ctx, messageKey)
	if err != nil {
		c.logger.Error("read idempotency state failed", "message_key", messageKey, "msg_id", msg.ID, "error", err)
		return
	}
	if state == "done" || state == "dead" {
		c.logger.Debug("skip already processed message", "message_key", messageKey, "msg_id", msg.ID, "state", state)
		c.ack(ctx, msg.ID)
		return
	}

	locked, err := c.acquireProcessingLock(ctx, messageKey)
	if err != nil {
		c.logger.Error("acquire processing lock failed", "message_key", messageKey, "msg_id", msg.ID, "error", err)
		return
	}
	if !locked {
		c.logger.Debug("skip duplicate message while another consumer is processing", "message_key", messageKey, "msg_id", msg.ID)
		c.ack(ctx, msg.ID)
		return
	}

	if strings.TrimSpace(eventMessage.Event) == "" {
		c.logger.Error("message missing event field", "msg_id", msg.ID)
		if err := c.sendToDeadLetter(ctx, msg, messageKey, "message missing event field", 0); err != nil {
			c.releaseProcessingLock(ctx, messageKey)
			return
		}
		if err := c.markDead(ctx, messageKey); err != nil {
			c.releaseProcessingLock(ctx, messageKey)
			return
		}
		c.releaseProcessingLock(ctx, messageKey)
		c.ack(ctx, msg.ID)
		return
	}

	handler, ok := c.handlers[eventMessage.Event]
	if !ok {
		c.logger.Error("no handler for event", "event", eventMessage.Event, "msg_id", msg.ID)
		if err := c.sendToDeadLetter(ctx, msg, messageKey, "no handler for event", 0); err != nil {
			c.releaseProcessingLock(ctx, messageKey)
			return
		}
		if err := c.markDead(ctx, messageKey); err != nil {
			c.releaseProcessingLock(ctx, messageKey)
			return
		}
		c.releaseProcessingLock(ctx, messageKey)
		c.ack(ctx, msg.ID)
		return
	}

	handlerCtx := ctx
	if eventMessage.TraceID != "" {
		handlerCtx = context.WithValue(ctx, ctxKey{}, eventMessage.TraceID)
	}

	if err := handler(handlerCtx, eventMessage); err != nil {
		retryCount, lookupErr := c.lookupRetryCount(ctx, msg.ID)
		if lookupErr != nil {
			c.logger.Error("lookup retry count failed", "event", eventMessage.Event, "msg_id", msg.ID, "error", lookupErr)
			c.releaseProcessingLock(ctx, messageKey)
			return
		}

		c.logger.Error("handler error", "event", eventMessage.Event, "error", err, "trace_id", eventMessage.TraceID, "retry_count", retryCount)
		if retryCount >= c.config.MaxRetries {
			if err := c.sendToDeadLetter(ctx, msg, messageKey, err.Error(), retryCount); err != nil {
				c.releaseProcessingLock(ctx, messageKey)
				return
			}
			if err := c.markDead(ctx, messageKey); err != nil {
				c.releaseProcessingLock(ctx, messageKey)
				return
			}
			c.releaseProcessingLock(ctx, messageKey)
			c.ack(ctx, msg.ID)
			return
		}
		c.releaseProcessingLock(ctx, messageKey)
		return
	}

	if err := c.markDone(ctx, messageKey); err != nil {
		c.logger.Error("mark message processed failed", "message_key", messageKey, "msg_id", msg.ID, "error", err)
		c.releaseProcessingLock(ctx, messageKey)
		return
	}
	c.releaseProcessingLock(ctx, messageKey)
	c.ack(ctx, msg.ID)
}

func (c *RedisConsumer) lookupRetryCount(ctx context.Context, messageID string) (int64, error) {
	entries, err := c.rdb.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: c.stream,
		Group:  c.group,
		Start:  messageID,
		End:    messageID,
		Count:  1,
	}).Result()
	if err != nil {
		return 0, err
	}
	if len(entries) == 0 {
		return 0, nil
	}
	return entries[0].RetryCount, nil
}

func (c *RedisConsumer) sendToDeadLetter(ctx context.Context, msg redis.XMessage, messageKey, reason string, retryCount int64) error {
	payload, _ := msg.Values["data"].(string)
	eventName, _ := msg.Values["event"].(string)
	traceID, _ := msg.Values["trace_id"].(string)

	vals := map[string]any{
		"message_key":         messageKey,
		"event":               eventName,
		"data":                payload,
		"original_message_id": msg.ID,
		"source_stream":       c.stream,
		"consumer_group":      c.group,
		"consumer":            c.consumer,
		"reason":              reason,
		"retry_count":         retryCount,
		"failed_at":           time.Now().UTC().Format(time.RFC3339),
	}
	if traceID != "" {
		vals["trace_id"] = traceID
	}

	if err := c.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: c.config.DeadLetterStream,
		Values: vals,
	}).Err(); err != nil {
		if c.logger != nil {
			c.logger.Error("write dead letter failed", "dead_letter_stream", c.config.DeadLetterStream, "msg_id", msg.ID, "error", err)
		}
		return err
	}

	if c.logger != nil {
		c.logger.Error(
			"message moved to dead letter stream",
			"dead_letter_stream", c.config.DeadLetterStream,
			"event", eventName,
			"msg_id", msg.ID,
			"retry_count", retryCount,
			"reason", reason,
		)
	}
	return nil
}

func (c *RedisConsumer) decodeMessage(msg redis.XMessage) Message {
	key, _ := msg.Values["message_key"].(string)
	eventName, _ := msg.Values["event"].(string)
	data, _ := msg.Values["data"].(string)
	traceID, _ := msg.Values["trace_id"].(string)

	return Message{
		Key:     key,
		Event:   eventName,
		Payload: []byte(data),
		TraceID: traceID,
	}
}

func (c *RedisConsumer) messageKey(msg redis.XMessage, message Message) string {
	if strings.TrimSpace(message.Key) != "" {
		return message.Key
	}
	return msg.ID
}

func (c *RedisConsumer) idempotencyState(ctx context.Context, messageKey string) (string, error) {
	value, err := c.rdb.Get(ctx, c.stateKey(messageKey)).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return value, err
}

func (c *RedisConsumer) acquireProcessingLock(ctx context.Context, messageKey string) (bool, error) {
	return c.rdb.SetNX(ctx, c.lockKey(messageKey), c.consumer, c.config.ProcessingLockTTL).Result()
}

func (c *RedisConsumer) releaseProcessingLock(ctx context.Context, messageKey string) {
	if err := c.rdb.Del(ctx, c.lockKey(messageKey)).Err(); err != nil && c.logger != nil {
		c.logger.Error("release processing lock failed", "message_key", messageKey, "error", err)
	}
}

func (c *RedisConsumer) markDone(ctx context.Context, messageKey string) error {
	return c.rdb.Set(ctx, c.stateKey(messageKey), "done", c.config.IdempotencyTTL).Err()
}

func (c *RedisConsumer) markDead(ctx context.Context, messageKey string) error {
	if err := c.rdb.Set(ctx, c.stateKey(messageKey), "dead", c.config.IdempotencyTTL).Err(); err != nil {
		if c.logger != nil {
			c.logger.Error("mark message dead failed", "message_key", messageKey, "error", err)
		}
		return err
	}
	return nil
}

func (c *RedisConsumer) stateKey(messageKey string) string {
	return "mq:consume:" + c.stream + ":" + c.group + ":" + messageKey + ":state"
}

func (c *RedisConsumer) lockKey(messageKey string) string {
	return "mq:consume:" + c.stream + ":" + c.group + ":" + messageKey + ":lock"
}

func (c *RedisConsumer) ack(ctx context.Context, id string) {
	if err := c.rdb.XAck(ctx, c.stream, c.group, id).Err(); err != nil {
		c.logger.Error("xack error", "error", err)
	}
}
