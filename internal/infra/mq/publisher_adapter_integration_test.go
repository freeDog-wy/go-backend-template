//go:build integration

package mq

import (
	"context"
	"testing"
	"time"

	"github.com/freeDog-wy/go-backend-template/internal/testsupport"
	pkgkafka "github.com/freeDog-wy/go-backend-template/pkg/kafka"
	"github.com/freeDog-wy/go-backend-template/pkg/logger"
	kgo "github.com/segmentio/kafka-go"
)

func TestPublisherAdapterIntegrationPublish(t *testing.T) {
	broker := testsupport.OpenKafka(t)
	topic := broker.CreateTopic(t, "integration.mq.publisher")
	publisher := newPublisherAdapter(broker.Brokers, topic, "integration-test", logger.Noop())
	message := Message{
		Key:     "message-key",
		Event:   "user.registered",
		Payload: []byte(`{"user_id":42}`),
		TraceID: "trace-integration-42",
	}
	if err := publisher.Publish(context.Background(), message); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	reader := pkgkafka.NewReader(broker.Brokers, topic, pkgkafka.ReaderConfig{
		ClientID:    "integration-reader",
		MinBytes:    1,
		MaxBytes:    1024,
		MaxWait:     100 * time.Millisecond,
		StartOffset: kgo.FirstOffset,
	})
	t.Cleanup(func() { _ = reader.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	record, err := reader.FetchMessage(ctx)
	if err != nil {
		t.Fatalf("FetchMessage() error = %v", err)
	}
	if string(record.Key) != message.Key {
		t.Fatalf("message key = %q, want %q", record.Key, message.Key)
	}
	if string(record.Value) != string(message.Payload) {
		t.Fatalf("payload = %q, want %q", record.Value, message.Payload)
	}
	if event := pkgkafka.HeaderValue(record.Headers, "event"); event != message.Event {
		t.Fatalf("event header = %q, want %q", event, message.Event)
	}
	if traceID := pkgkafka.HeaderValue(record.Headers, traceIDHeader); traceID != message.TraceID {
		t.Fatalf("trace header = %q, want %q", traceID, message.TraceID)
	}
}
