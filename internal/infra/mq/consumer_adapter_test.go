package mq

import (
	"context"
	"errors"
	"testing"
	"time"

	pkgkafka "github.com/freeDog-wy/go-backend-template/pkg/kafka"
)

func TestDecodeMessage(t *testing.T) {
	withTestPropagator(t)

	adapter := &consumerAdapter{}

	t.Run("decodes message with explicit trace id and trace context", func(t *testing.T) {
		headers := append(
			[]pkgkafka.Header{
				{Key: "event", Value: []byte("user.registered")},
				{Key: traceIDHeader, Value: []byte("legacy-trace-id")},
			},
			InjectTraceContext(newTraceContext(t), nil)...,
		)

		message, err := adapter.decodeMessage(pkgkafka.Record{
			Message: pkgkafka.Message{
				Key:     []byte("message-1"),
				Value:   []byte(`{"id":1}`),
				Headers: headers,
			},
		})
		if err != nil {
			t.Fatalf("decodeMessage() error = %v", err)
		}
		if message.Key != "message-1" {
			t.Fatalf("key = %q, want %q", message.Key, "message-1")
		}
		if message.Event != "user.registered" {
			t.Fatalf("event = %q, want %q", message.Event, "user.registered")
		}
		if string(message.Payload) != `{"id":1}` {
			t.Fatalf("payload = %q", string(message.Payload))
		}
		if message.TraceID != "legacy-trace-id" {
			t.Fatalf("trace id = %q, want %q", message.TraceID, "legacy-trace-id")
		}
		if message.TraceContext == "" {
			t.Fatal("trace context should not be empty")
		}
	})

	t.Run("falls back to traceparent when trace_id is absent", func(t *testing.T) {
		headers := append(
			[]pkgkafka.Header{{Key: "event", Value: []byte("user.registered")}},
			InjectTraceContext(newTraceContext(t), nil)...,
		)

		message, err := adapter.decodeMessage(pkgkafka.Record{
			Message: pkgkafka.Message{
				Key:     []byte("message-1"),
				Value:   []byte(`{"id":1}`),
				Headers: headers,
			},
		})
		if err != nil {
			t.Fatalf("decodeMessage() error = %v", err)
		}
		if message.TraceID != "00112233445566778899aabbccddeeff" {
			t.Fatalf("trace id = %q", message.TraceID)
		}
	})

	t.Run("returns error when event header is missing", func(t *testing.T) {
		_, err := adapter.decodeMessage(pkgkafka.Record{
			Message: pkgkafka.Message{
				Key:   []byte("message-1"),
				Value: []byte(`{"id":1}`),
			},
		})
		if err == nil || err.Error() != "message missing event header" {
			t.Fatalf("decodeMessage() error = %v", err)
		}
	})

	t.Run("returns error when message key is missing", func(t *testing.T) {
		_, err := adapter.decodeMessage(pkgkafka.Record{
			Message: pkgkafka.Message{
				Key:     []byte("   "),
				Value:   []byte(`{"id":1}`),
				Headers: []pkgkafka.Header{{Key: "event", Value: []byte("user.registered")}},
			},
		})
		if err == nil || err.Error() != "message missing key" {
			t.Fatalf("decodeMessage() error = %v", err)
		}
	})
}

func TestWaitRetryDelay(t *testing.T) {
	t.Run("returns immediately when no delay is configured", func(t *testing.T) {
		adapter := &consumerAdapter{}
		start := time.Now()
		err := adapter.waitRetryDelay(context.Background(), pkgkafka.ReaderLoop{}, Message{})
		if err != nil {
			t.Fatalf("waitRetryDelay() error = %v", err)
		}
		if time.Since(start) > 50*time.Millisecond {
			t.Fatal("waitRetryDelay() should return immediately when delay <= 0")
		}
	})

	t.Run("returns context error when context is canceled before delay elapses", func(t *testing.T) {
		adapter := &consumerAdapter{}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := adapter.waitRetryDelay(ctx, pkgkafka.ReaderLoop{Delay: 100 * time.Millisecond}, Message{})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("waitRetryDelay() error = %v, want %v", err, context.Canceled)
		}
	})

	t.Run("waits for configured delay", func(t *testing.T) {
		adapter := &consumerAdapter{}
		delay := 20 * time.Millisecond
		start := time.Now()

		err := adapter.waitRetryDelay(context.Background(), pkgkafka.ReaderLoop{Delay: delay}, Message{
			Event: "user.registered",
			Key:   "message-1",
		})
		if err != nil {
			t.Fatalf("waitRetryDelay() error = %v", err)
		}
		if elapsed := time.Since(start); elapsed < delay {
			t.Fatalf("waitRetryDelay() elapsed = %v, want >= %v", elapsed, delay)
		}
	})
}

func TestRetryRoute(t *testing.T) {
	t.Parallel()

	adapter := &consumerAdapter{
		topology: &pkgkafka.ConsumerTopology{
			RetryPublishers: []pkgkafka.RetryPublisher{
				{Topic: "retry.30s", Delay: 30 * time.Second},
				{Topic: "retry.5m", Delay: 5 * time.Minute},
				{Topic: "retry.30m", Delay: 30 * time.Minute},
			},
		},
	}

	tests := []struct {
		name         string
		attemptCount int
		wantTopic    string
	}{
		{name: "clamps negative attempt to first retry topic", attemptCount: -1, wantTopic: "retry.30s"},
		{name: "maps first attempt to first retry topic", attemptCount: 1, wantTopic: "retry.30s"},
		{name: "maps second attempt to second retry topic", attemptCount: 2, wantTopic: "retry.5m"},
		{name: "clamps overflow attempt to last retry topic", attemptCount: 99, wantTopic: "retry.30m"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			route := adapter.retryRoute(tt.attemptCount)
			if route.Topic != tt.wantTopic {
				t.Fatalf("retryRoute(%d) topic = %q, want %q", tt.attemptCount, route.Topic, tt.wantTopic)
			}
		})
	}
}

func TestOriginalTopic(t *testing.T) {
	t.Parallel()

	record := pkgkafka.Record{
		Message: pkgkafka.Message{
			Headers: []pkgkafka.Header{
				{Key: "original_topic", Value: []byte("domain.events")},
			},
		},
		Topic: "domain.events.retry.30s",
	}
	if got := originalTopic(record); got != "domain.events" {
		t.Fatalf("originalTopic() = %q, want %q", got, "domain.events")
	}

	record.Headers = nil
	if got := originalTopic(record); got != "domain.events.retry.30s" {
		t.Fatalf("originalTopic() fallback = %q, want %q", got, "domain.events.retry.30s")
	}
}
