package mq

import (
	"context"

	"github.com/freeDog-wy/go-backend-template/pkg/logger"

	"github.com/redis/go-redis/v9"
)

// RedisPublisher 只负责把统一消息模型投递到 Redis Streams。
type RedisPublisher struct {
	rdb    *redis.Client
	stream string
	logger logger.Logger
}

func NewRedisPublisher(rdb *redis.Client, stream string, log logger.Logger) *RedisPublisher {
	return &RedisPublisher{rdb: rdb, stream: stream, logger: log}
}

var _ Publisher = (*RedisPublisher)(nil)

// Publish 不关心业务来源，只负责把统一消息结构映射到 Redis 字段。
func (p *RedisPublisher) Publish(ctx context.Context, message Message) error {
	vals := map[string]any{
		"event": message.Event,
		"data":  string(message.Payload),
	}
	if message.Key != "" {
		vals["message_key"] = message.Key
	}
	if message.TraceID != "" {
		vals["trace_id"] = message.TraceID
	}

	if err := p.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: p.stream,
		Values: vals,
	}).Err(); err != nil {
		return err
	}

	if p.logger != nil {
		p.logger.Debug("event published", "event", message.Event, "trace_id", message.TraceID)
	}
	return nil
}
