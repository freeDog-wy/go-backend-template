package mq

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	domainConsumption "github.com/freeDog-wy/go-backend-template/internal/domain/consumption"
	"github.com/freeDog-wy/go-backend-template/pkg/logger"

	"github.com/segmentio/kafka-go"
)

type KafkaConsumerConfig struct {
	GroupID           string
	ClientID          string
	MinBytes          int
	MaxBytes          int
	MaxWait           time.Duration
	ProcessingLockTTL time.Duration
	MaxRetries        int
	RetryTopic        string
	DeadLetterTopic   string
}

// KafkaConsumer 从 Kafka topic 读取消息，并分发给注册的 handler。
// 当前策略：
// 1. 成功处理后提交 offset
// 2. 可重试错误转发到 retry topic
// 3. 超过最大次数或不可重试错误转发到 DLQ topic
type KafkaConsumer struct {
	reader             *kafka.Reader
	retryWriter        *kafka.Writer
	deadLetterWriter   *kafka.Writer
	topics             []string
	groupID            string
	retryTopic         string
	deadLetterTopic    string
	processingLockTTL  time.Duration
	maxRetries         int
	consumptionRecords domainConsumption.Repository
	handlers           map[string]EventHandler
	logger             logger.Logger
}

var _ Consumer = (*KafkaConsumer)(nil)

func NewKafkaConsumer(
	brokers []string,
	topic string,
	log logger.Logger,
	records domainConsumption.Repository,
	cfg KafkaConsumerConfig,
) *KafkaConsumer {
	normalizedBrokers := normalizeKafkaBrokers(brokers)
	if len(normalizedBrokers) == 0 {
		panic("kafka brokers must not be empty")
	}
	if strings.TrimSpace(topic) == "" {
		panic("kafka topic must not be empty")
	}
	if strings.TrimSpace(cfg.GroupID) == "" {
		panic("kafka consumer group id must not be empty")
	}
	if cfg.MinBytes <= 0 {
		cfg.MinBytes = 1024
	}
	if cfg.MaxBytes <= 0 {
		cfg.MaxBytes = 10 * 1024 * 1024
	}
	if cfg.MaxWait <= 0 {
		cfg.MaxWait = time.Second
	}
	if cfg.ProcessingLockTTL <= 0 {
		cfg.ProcessingLockTTL = 5 * time.Minute
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 10
	}
	if records == nil {
		panic("kafka consumption repository must not be nil")
	}

	mainTopic := strings.TrimSpace(topic)
	retryTopic := strings.TrimSpace(cfg.RetryTopic)
	if retryTopic == "" {
		retryTopic = mainTopic + ".retry"
	}
	deadLetterTopic := strings.TrimSpace(cfg.DeadLetterTopic)
	if deadLetterTopic == "" {
		deadLetterTopic = mainTopic + ".dlq"
	}

	topics := uniqueTopics(mainTopic, retryTopic)
	readerConfig := kafka.ReaderConfig{
		Brokers:        normalizedBrokers,
		GroupID:        cfg.GroupID,
		MinBytes:       cfg.MinBytes,
		MaxBytes:       cfg.MaxBytes,
		MaxWait:        cfg.MaxWait,
		CommitInterval: 0,
		StartOffset:    kafka.LastOffset,
		Dialer: &kafka.Dialer{
			ClientID: strings.TrimSpace(cfg.ClientID),
		},
	}
	if len(topics) == 1 {
		readerConfig.Topic = topics[0]
	} else {
		readerConfig.GroupTopics = topics
	}

	return &KafkaConsumer{
		reader:             kafka.NewReader(readerConfig),
		retryWriter:        newKafkaWriter(normalizedBrokers, retryTopic, cfg.ClientID),
		deadLetterWriter:   newKafkaWriter(normalizedBrokers, deadLetterTopic, cfg.ClientID),
		topics:             topics,
		groupID:            cfg.GroupID,
		retryTopic:         retryTopic,
		deadLetterTopic:    deadLetterTopic,
		processingLockTTL:  cfg.ProcessingLockTTL,
		maxRetries:         cfg.MaxRetries,
		consumptionRecords: records,
		handlers:           make(map[string]EventHandler),
		logger:             log,
	}
}

func (c *KafkaConsumer) Handle(eventName string, fn EventHandler) {
	c.handlers[eventName] = fn
}

func (c *KafkaConsumer) Run(ctx context.Context) error {
	c.logger.Info(
		"consumer started",
		"group", c.groupID,
		"topics", strings.Join(c.topics, ","),
		"retry_topic", c.retryTopic,
		"dead_letter_topic", c.deadLetterTopic,
		"provider", "kafka",
	)

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return ctx.Err()
			}
			return err
		}

		eventMessage, err := c.decodeMessage(msg)
		if err != nil {
			if routeErr := c.routeMalformedToDeadLetter(ctx, msg, err); routeErr != nil {
				return routeErr
			}
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				c.logger.Error("commit malformed kafka message failed", "topic", msg.Topic, "partition", msg.Partition, "offset", msg.Offset, "error", err)
				return err
			}
			continue
		}

		beginAt := time.Now()
		beginResult, err := c.beginConsumption(ctx, eventMessage, beginAt)
		if err != nil {
			c.logger.Error("begin kafka consumption failed", "event", eventMessage.Event, "message_key", eventMessage.Key, "error", err)
			return err
		}

		switch beginResult.Decision {
		case domainConsumption.BeginDecisionDone:
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				c.logger.Error("commit already processed kafka message failed", "event", eventMessage.Event, "message_key", eventMessage.Key, "offset", msg.Offset, "error", err)
				return err
			}
			continue
		case domainConsumption.BeginDecisionLocked:
			err := errors.New("message is being processed by another worker")
			c.logger.Error("kafka message processing lock is active", "event", eventMessage.Event, "message_key", eventMessage.Key, "offset", msg.Offset)
			return err
		}

		handlerCtx := ctx
		if eventMessage.TraceID != "" {
			handlerCtx = context.WithValue(ctx, ctxKey{}, eventMessage.TraceID)
		}

		handler, ok := c.handlers[eventMessage.Event]
		if !ok {
			handlerErr := MarkNonRetryable(errors.New("no handler for event: " + eventMessage.Event))
			if err := c.handleFailure(ctx, msg, eventMessage, beginResult.AttemptCount, handlerErr); err != nil {
				return err
			}
			continue
		}

		if err := handler(handlerCtx, eventMessage); err != nil {
			if err := c.handleFailure(ctx, msg, eventMessage, beginResult.AttemptCount, err); err != nil {
				return err
			}
			continue
		}

		if err := c.markDone(ctx, eventMessage.Key, time.Now()); err != nil {
			c.logger.Error("mark kafka message done state error", "event", eventMessage.Event, "message_key", eventMessage.Key, "error", err)
			return err
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.Error("commit kafka message failed", "event", eventMessage.Event, "topic", msg.Topic, "partition", msg.Partition, "offset", msg.Offset, "error", err)
			return err
		}
	}
}

func (c *KafkaConsumer) decodeMessage(msg kafka.Message) (Message, error) {
	eventName := headerValue(msg.Headers, "event")
	if strings.TrimSpace(eventName) == "" {
		return Message{}, errors.New("message missing event header")
	}

	messageKey := strings.TrimSpace(string(msg.Key))
	if messageKey == "" {
		return Message{}, errors.New("message missing key")
	}

	return Message{
		Key:     messageKey,
		Event:   eventName,
		Payload: msg.Value,
		TraceID: headerValue(msg.Headers, "trace_id"),
	}, nil
}

func (c *KafkaConsumer) handleFailure(ctx context.Context, msg kafka.Message, message Message, attemptCount int, handlerErr error) error {
	now := time.Now()
	nonRetryable := IsNonRetryable(handlerErr)
	if nonRetryable || attemptCount >= c.maxRetries {
		if err := c.publishDeadLetter(ctx, msg, message, attemptCount, handlerErr); err != nil {
			return err
		}
		if err := c.markDead(ctx, message.Key, handlerErr, now); err != nil {
			c.logger.Error("mark kafka message dead state error", "event", message.Event, "message_key", message.Key, "error", err)
			return err
		}
		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.Error("commit kafka dead letter message failed", "event", message.Event, "topic", msg.Topic, "partition", msg.Partition, "offset", msg.Offset, "error", err)
			return err
		}
		c.logger.Error("kafka message moved to dead letter topic", "event", message.Event, "message_key", message.Key, "attempt_count", attemptCount, "error", handlerErr)
		return nil
	}

	if err := c.publishRetry(ctx, msg, message, attemptCount, handlerErr); err != nil {
		return err
	}
	if err := c.markFailed(ctx, message.Key, handlerErr, now); err != nil {
		c.logger.Error("mark kafka message failed state error", "event", message.Event, "message_key", message.Key, "error", err)
		return err
	}
	if err := c.reader.CommitMessages(ctx, msg); err != nil {
		c.logger.Error("commit kafka retried message failed", "event", message.Event, "topic", msg.Topic, "partition", msg.Partition, "offset", msg.Offset, "error", err)
		return err
	}
	c.logger.Error("kafka message moved to retry topic", "event", message.Event, "message_key", message.Key, "attempt_count", attemptCount, "error", handlerErr)
	return nil
}

func (c *KafkaConsumer) routeMalformedToDeadLetter(ctx context.Context, msg kafka.Message, decodeErr error) error {
	if err := c.deadLetterWriter.WriteMessages(ctx, kafka.Message{
		Key:   msg.Key,
		Value: msg.Value,
		Headers: appendKafkaHeaders(
			msg.Headers,
			kafka.Header{Key: "reason", Value: []byte(strings.TrimSpace(decodeErr.Error()))},
			kafka.Header{Key: "failed_at", Value: []byte(time.Now().UTC().Format(time.RFC3339))},
			kafka.Header{Key: "consumer_group", Value: []byte(c.groupID)},
			kafka.Header{Key: "source_topic", Value: []byte(msg.Topic)},
			kafka.Header{Key: "source_partition", Value: []byte(strconv.Itoa(msg.Partition))},
			kafka.Header{Key: "source_offset", Value: []byte(strconv.FormatInt(msg.Offset, 10))},
		),
	}); err != nil {
		c.logger.Error("write malformed kafka dead letter failed", "topic", msg.Topic, "offset", msg.Offset, "error", err)
		return err
	}
	c.logger.Error("malformed kafka message moved to dead letter topic", "topic", msg.Topic, "offset", msg.Offset, "error", decodeErr)
	return nil
}

func (c *KafkaConsumer) publishRetry(ctx context.Context, msg kafka.Message, message Message, attemptCount int, handlerErr error) error {
	retryMessage := kafka.Message{
		Key:   []byte(message.Key),
		Value: message.Payload,
		Headers: []kafka.Header{
			{Key: "event", Value: []byte(message.Event)},
			{Key: "trace_id", Value: []byte(message.TraceID)},
			{Key: "retry_count", Value: []byte(strconv.Itoa(attemptCount))},
			{Key: "original_topic", Value: []byte(originalTopic(msg))},
			{Key: "source_topic", Value: []byte(msg.Topic)},
			{Key: "source_partition", Value: []byte(strconv.Itoa(msg.Partition))},
			{Key: "source_offset", Value: []byte(strconv.FormatInt(msg.Offset, 10))},
			{Key: "last_error", Value: []byte(strings.TrimSpace(handlerErr.Error()))},
		},
	}
	if err := c.retryWriter.WriteMessages(ctx, retryMessage); err != nil {
		c.logger.Error("write kafka retry message failed", "event", message.Event, "message_key", message.Key, "error", err)
		return err
	}
	return nil
}

func (c *KafkaConsumer) publishDeadLetter(ctx context.Context, msg kafka.Message, message Message, attemptCount int, handlerErr error) error {
	deadLetterMessage := kafka.Message{
		Key:   []byte(message.Key),
		Value: message.Payload,
		Headers: []kafka.Header{
			{Key: "event", Value: []byte(message.Event)},
			{Key: "trace_id", Value: []byte(message.TraceID)},
			{Key: "retry_count", Value: []byte(strconv.Itoa(attemptCount))},
			{Key: "original_topic", Value: []byte(originalTopic(msg))},
			{Key: "source_topic", Value: []byte(msg.Topic)},
			{Key: "source_partition", Value: []byte(strconv.Itoa(msg.Partition))},
			{Key: "source_offset", Value: []byte(strconv.FormatInt(msg.Offset, 10))},
			{Key: "reason", Value: []byte(strings.TrimSpace(handlerErr.Error()))},
			{Key: "failed_at", Value: []byte(time.Now().UTC().Format(time.RFC3339))},
			{Key: "consumer_group", Value: []byte(c.groupID)},
		},
	}
	if err := c.deadLetterWriter.WriteMessages(ctx, deadLetterMessage); err != nil {
		c.logger.Error("write kafka dead letter message failed", "event", message.Event, "message_key", message.Key, "error", err)
		return err
	}
	return nil
}

func (c *KafkaConsumer) beginConsumption(ctx context.Context, message Message, attemptedAt time.Time) (domainConsumption.BeginResult, error) {
	return c.consumptionRecords.Begin(ctx, domainConsumption.BeginCommand{
		ConsumerGroup: c.groupID,
		MessageKey:    message.Key,
		EventName:     message.Event,
		TraceID:       message.TraceID,
		AttemptedAt:   attemptedAt,
		LockedUntil:   attemptedAt.Add(c.processingLockTTL),
	})
}

func (c *KafkaConsumer) markDone(ctx context.Context, messageKey string, processedAt time.Time) error {
	return c.consumptionRecords.MarkDone(ctx, c.groupID, messageKey, processedAt)
}

func (c *KafkaConsumer) markFailed(ctx context.Context, messageKey string, handlerErr error, failedAt time.Time) error {
	if handlerErr == nil {
		return nil
	}
	return c.consumptionRecords.MarkFailed(ctx, c.groupID, messageKey, handlerErr.Error(), failedAt)
}

func (c *KafkaConsumer) markDead(ctx context.Context, messageKey string, handlerErr error, failedAt time.Time) error {
	if handlerErr == nil {
		return nil
	}
	return c.consumptionRecords.MarkDead(ctx, c.groupID, messageKey, handlerErr.Error(), failedAt)
}

func headerValue(headers []kafka.Header, key string) string {
	for _, header := range headers {
		if header.Key == key {
			return string(header.Value)
		}
	}
	return ""
}

func appendKafkaHeaders(headers []kafka.Header, extras ...kafka.Header) []kafka.Header {
	result := make([]kafka.Header, 0, len(headers)+len(extras))
	result = append(result, headers...)
	result = append(result, extras...)
	return result
}

func originalTopic(msg kafka.Message) string {
	if value := headerValue(msg.Headers, "original_topic"); strings.TrimSpace(value) != "" {
		return value
	}
	return msg.Topic
}

func uniqueTopics(values ...string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}
