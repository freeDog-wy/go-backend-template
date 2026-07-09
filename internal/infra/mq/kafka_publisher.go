package mq

import (
	"context"

	"github.com/freeDog-wy/go-backend-template/pkg/logger"

	"github.com/segmentio/kafka-go"
)

// KafkaPublisher 把统一消息模型投递到 Kafka topic。
type KafkaPublisher struct {
	writer *kafka.Writer
	logger logger.Logger
}

func NewKafkaPublisher(brokers []string, topic, clientID string, log logger.Logger) *KafkaPublisher {
	return &KafkaPublisher{
		writer: newKafkaWriter(brokers, topic, clientID),
		logger: log,
	}
}

var _ Publisher = (*KafkaPublisher)(nil)

// Publish 把统一消息结构映射到 Kafka message。
func (p *KafkaPublisher) Publish(ctx context.Context, message Message) error {
	headers := []kafka.Header{
		{Key: "event", Value: []byte(message.Event)},
	}
	if message.TraceID != "" {
		headers = append(headers, kafka.Header{Key: "trace_id", Value: []byte(message.TraceID)})
	}

	msg := kafka.Message{
		Key:     []byte(message.Key),
		Value:   message.Payload,
		Headers: headers,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return err
	}

	if p.logger != nil {
		p.logger.Debug("event published", "event", message.Event, "trace_id", message.TraceID)
	}
	return nil
}
