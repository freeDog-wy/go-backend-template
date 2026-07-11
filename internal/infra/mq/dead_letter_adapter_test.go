package mq

import (
	"context"
	"strings"
	"testing"
	"time"

	pkgkafka "github.com/freeDog-wy/go-backend-template/pkg/kafka"
)

func TestDecodeDeadLetter(t *testing.T) {
	withTestPropagator(t)

	adapter := &deadLetterAdapter{}
	headers := deadLetterHeaders(newTraceHeaders(t))
	record := pkgkafka.DeadLetterRecord{
		Key:       []byte("message-1"),
		Value:     []byte(`{"hello":"world"}`),
		Headers:   headers,
		Topic:     "domain.events.dlq",
		Partition: 2,
		Offset:    99,
	}

	t.Run("decodes complete dead letter message", func(t *testing.T) {
		t.Parallel()

		message, err := adapter.decodeDeadLetter(record)
		if err != nil {
			t.Fatalf("decodeDeadLetter() error = %v", err)
		}
		if message.Key != "message-1" {
			t.Fatalf("message key = %q, want %q", message.Key, "message-1")
		}
		if message.Event != "user.registered" {
			t.Fatalf("event = %q, want %q", message.Event, "user.registered")
		}
		if message.TraceID != "legacy-trace-id" {
			t.Fatalf("trace id = %q, want %q", message.TraceID, "legacy-trace-id")
		}
		if message.TraceContext == "" {
			t.Fatal("trace context should not be empty")
		}
		if message.OriginalMessageID != "outbox-1" {
			t.Fatalf("original message id = %q", message.OriginalMessageID)
		}
		if message.OriginalTopic != "domain.events" {
			t.Fatalf("original topic = %q, want %q", message.OriginalTopic, "domain.events")
		}
		if message.Source != "domain.events.retry.30s" {
			t.Fatalf("source = %q, want %q", message.Source, "domain.events.retry.30s")
		}
		if message.SourcePartition != 1 {
			t.Fatalf("source partition = %d, want %d", message.SourcePartition, 1)
		}
		if message.SourceOffset != 42 {
			t.Fatalf("source offset = %d, want %d", message.SourceOffset, 42)
		}
		if message.ConsumerGroup != "user-worker" {
			t.Fatalf("consumer group = %q, want %q", message.ConsumerGroup, "user-worker")
		}
		if message.Reason != "handler failed" {
			t.Fatalf("reason = %q, want %q", message.Reason, "handler failed")
		}
		if message.RetryCount != 3 {
			t.Fatalf("retry count = %d, want %d", message.RetryCount, 3)
		}
		if message.RetryTopic != "domain.events.retry.30s" {
			t.Fatalf("retry topic = %q, want %q", message.RetryTopic, "domain.events.retry.30s")
		}
		if message.RetryDelaySeconds != 30 {
			t.Fatalf("retry delay seconds = %d, want %d", message.RetryDelaySeconds, 30)
		}
		if message.FailedAt.IsZero() {
			t.Fatal("failed at should not be zero")
		}
		if message.DeadLetterTopic != "domain.events.dlq" {
			t.Fatalf("dead letter topic = %q, want %q", message.DeadLetterTopic, "domain.events.dlq")
		}
		if message.DeadLetterOffset != 99 {
			t.Fatalf("dead letter offset = %d, want %d", message.DeadLetterOffset, 99)
		}
		if message.DeadLetterPart != 2 {
			t.Fatalf("dead letter partition = %d, want %d", message.DeadLetterPart, 2)
		}
	})

	t.Run("returns error when event header is missing", func(t *testing.T) {
		t.Parallel()

		badRecord := record
		badRecord.Headers = removeHeader(headers, "event")

		_, err := adapter.decodeDeadLetter(badRecord)
		if err == nil || err.Error() != "dead letter message missing event header" {
			t.Fatalf("decodeDeadLetter() error = %v", err)
		}
	})

	t.Run("returns error when key is missing", func(t *testing.T) {
		t.Parallel()

		badRecord := record
		badRecord.Key = []byte("   ")

		_, err := adapter.decodeDeadLetter(badRecord)
		if err == nil || err.Error() != "dead letter message missing key" {
			t.Fatalf("decodeDeadLetter() error = %v", err)
		}
	})

	t.Run("returns parse error for invalid retry_count", func(t *testing.T) {
		t.Parallel()

		badRecord := record
		badRecord.Headers = replaceHeader(headers, "retry_count", "abc")

		_, err := adapter.decodeDeadLetter(badRecord)
		if err == nil || !strings.Contains(err.Error(), "invalid kafka header retry_count") {
			t.Fatalf("decodeDeadLetter() error = %v", err)
		}
	})
}

func TestBuildReplayMessage(t *testing.T) {
	withTestPropagator(t)

	adapter := &deadLetterAdapter{}
	headers := deadLetterHeaders(newTraceHeaders(t))
	record := pkgkafka.DeadLetterRecord{
		Key:       []byte("message-1"),
		Value:     []byte(`{"hello":"world"}`),
		Headers:   headers,
		Topic:     "domain.events.dlq",
		Partition: 2,
		Offset:    99,
	}

	message, err := adapter.buildReplayMessage(record)
	if err != nil {
		t.Fatalf("buildReplayMessage() error = %v", err)
	}

	if got := string(message.Key); !strings.HasPrefix(got, "dlq-replay-") {
		t.Fatalf("replay message key = %q", got)
	}
	if string(message.Value) != `{"hello":"world"}` {
		t.Fatalf("message value = %q", string(message.Value))
	}

	assertHeaderValue(t, message.Headers, "event", "user.registered")
	assertHeaderValue(t, message.Headers, "replayed_from_dlq", "true")
	assertHeaderValue(t, message.Headers, "original_message_key", "message-1")
	assertHeaderValue(t, message.Headers, traceIDHeader, "legacy-trace-id")
	assertHeaderValue(t, message.Headers, "original_topic", "domain.events")
	assertHeaderValue(t, message.Headers, "dlq_topic", "domain.events.dlq")
	assertHeaderValue(t, message.Headers, "dlq_reason", "handler failed")
	assertHeaderValue(t, message.Headers, "dlq_original_source_topic", "domain.events.retry.30s")
	assertHeaderValue(t, message.Headers, "dlq_source_partition", "2")
	assertHeaderValue(t, message.Headers, "dlq_source_offset", "99")

	if got := pkgkafka.HeaderValue(message.Headers, "dlq_replayed_at"); got == "" {
		t.Fatal("dlq_replayed_at header should not be empty")
	}
	if got := pkgkafka.HeaderValue(message.Headers, "dlq_failed_at"); got == "" {
		t.Fatal("dlq_failed_at header should not be empty")
	}
	if got := pkgkafka.HeaderValue(message.Headers, "traceparent"); got == "" {
		t.Fatal("traceparent header should be preserved on replay")
	}
}

func TestBuildReplayMessageOmitsOptionalHeadersWhenAbsent(t *testing.T) {
	withTestPropagator(t)

	adapter := &deadLetterAdapter{}
	record := pkgkafka.DeadLetterRecord{
		Key:   []byte("message-1"),
		Value: []byte(`{"hello":"world"}`),
		Headers: []pkgkafka.Header{
			{Key: "event", Value: []byte("user.registered")},
		},
		Topic:     "domain.events.dlq",
		Partition: -1,
		Offset:    -1,
	}

	message, err := adapter.buildReplayMessage(record)
	if err != nil {
		t.Fatalf("buildReplayMessage() error = %v", err)
	}

	if got := pkgkafka.HeaderValue(message.Headers, traceIDHeader); got != "" {
		t.Fatalf("trace_id header = %q, want empty", got)
	}
	if got := pkgkafka.HeaderValue(message.Headers, "original_topic"); got != "domain.events.dlq" {
		t.Fatalf("original_topic header = %q, want %q", got, "domain.events.dlq")
	}
	if got := pkgkafka.HeaderValue(message.Headers, "dlq_reason"); got != "" {
		t.Fatalf("dlq_reason header = %q, want empty", got)
	}
	if got := pkgkafka.HeaderValue(message.Headers, "dlq_source_partition"); got != "" {
		t.Fatalf("dlq_source_partition header = %q, want empty", got)
	}
	if got := pkgkafka.HeaderValue(message.Headers, "traceparent"); got != "" {
		t.Fatalf("traceparent header = %q, want empty", got)
	}
}

func TestNewReplayMessageKey(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 7, 11, 12, 0, 0, 123, time.UTC)
	key1 := newReplayMessageKey("message-1", at)
	key2 := newReplayMessageKey("message-1", at)
	key3 := newReplayMessageKey("message-2", at)

	if key1 != key2 {
		t.Fatalf("same input should produce same key: %q != %q", key1, key2)
	}
	if key1 == key3 {
		t.Fatalf("different input should produce different key: %q == %q", key1, key3)
	}
	if !strings.HasPrefix(key1, "dlq-replay-") {
		t.Fatalf("key prefix = %q", key1)
	}
}

func newTraceHeaders(t *testing.T) []pkgkafka.Header {
	t.Helper()

	return append(
		[]pkgkafka.Header{{Key: traceIDHeader, Value: []byte("legacy-trace-id")}},
		InjectTraceContext(newTraceContext(t), nil)...,
	)
}

func deadLetterHeaders(base []pkgkafka.Header) []pkgkafka.Header {
	headers := append([]pkgkafka.Header(nil), base...)
	headers = append(headers,
		pkgkafka.Header{Key: "event", Value: []byte("user.registered")},
		pkgkafka.Header{Key: "original_message_id", Value: []byte("outbox-1")},
		pkgkafka.Header{Key: "original_topic", Value: []byte("domain.events")},
		pkgkafka.Header{Key: "source_topic", Value: []byte("domain.events.retry.30s")},
		pkgkafka.Header{Key: "source_partition", Value: []byte("1")},
		pkgkafka.Header{Key: "source_offset", Value: []byte("42")},
		pkgkafka.Header{Key: "consumer_group", Value: []byte("user-worker")},
		pkgkafka.Header{Key: "consumer", Value: []byte("worker-1")},
		pkgkafka.Header{Key: "reason", Value: []byte("handler failed")},
		pkgkafka.Header{Key: "retry_count", Value: []byte("3")},
		pkgkafka.Header{Key: "retry_topic", Value: []byte("domain.events.retry.30s")},
		pkgkafka.Header{Key: "retry_delay_seconds", Value: []byte("30")},
		pkgkafka.Header{Key: "failed_at", Value: []byte("2026-07-11T10:20:30Z")},
	)
	return headers
}

func removeHeader(headers []pkgkafka.Header, key string) []pkgkafka.Header {
	result := make([]pkgkafka.Header, 0, len(headers))
	for _, header := range headers {
		if header.Key == key {
			continue
		}
		result = append(result, header)
	}
	return result
}

func replaceHeader(headers []pkgkafka.Header, key, value string) []pkgkafka.Header {
	result := append([]pkgkafka.Header(nil), headers...)
	for i := range result {
		if result[i].Key == key {
			result[i].Value = []byte(value)
			return result
		}
	}
	return append(result, pkgkafka.Header{Key: key, Value: []byte(value)})
}

func assertHeaderValue(t *testing.T, headers []pkgkafka.Header, key, want string) {
	t.Helper()

	if got := pkgkafka.HeaderValue(headers, key); got != want {
		t.Fatalf("header %s = %q, want %q", key, got, want)
	}
}

func TestOriginalTopicFromRecord(t *testing.T) {
	t.Parallel()

	record := pkgkafka.DeadLetterRecord{
		Topic: "domain.events.dlq",
		Headers: []pkgkafka.Header{
			{Key: "original_topic", Value: []byte("domain.events")},
		},
	}
	if got := originalTopicFromRecord(record); got != "domain.events" {
		t.Fatalf("originalTopicFromRecord() = %q, want %q", got, "domain.events")
	}

	record.Headers = nil
	if got := originalTopicFromRecord(record); got != "domain.events.dlq" {
		t.Fatalf("originalTopicFromRecord() fallback = %q, want %q", got, "domain.events.dlq")
	}
}

func TestBuildReplayMessageReturnsDecodeError(t *testing.T) {
	t.Parallel()

	adapter := &deadLetterAdapter{}
	_, err := adapter.buildReplayMessage(pkgkafka.DeadLetterRecord{
		Key: []byte("message-1"),
	})
	if err == nil || err.Error() != "dead letter message missing event header" {
		t.Fatalf("buildReplayMessage() error = %v", err)
	}
}

func TestReplayBuildMessagePreservesTraceContextRoundTrip(t *testing.T) {
	withTestPropagator(t)

	adapter := &deadLetterAdapter{}
	record := pkgkafka.DeadLetterRecord{
		Key:       []byte("message-1"),
		Value:     []byte(`{"hello":"world"}`),
		Headers:   deadLetterHeaders(newTraceHeaders(t)),
		Topic:     "domain.events.dlq",
		Partition: 0,
		Offset:    1,
	}

	message, err := adapter.buildReplayMessage(record)
	if err != nil {
		t.Fatalf("buildReplayMessage() error = %v", err)
	}

	ctx := ExtractTraceContext(context.Background(), message.Headers)
	if got := TraceIDFromContext(ctx); got != "00112233445566778899aabbccddeeff" {
		t.Fatalf("TraceIDFromContext(replayed headers) = %q", got)
	}
}
