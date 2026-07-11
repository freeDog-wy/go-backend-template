package mq

import (
	"context"
	"testing"

	pkgkafka "github.com/freeDog-wy/go-backend-template/pkg/kafka"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestTraceIDFromContext(t *testing.T) {
	withTestPropagator(t)

	traceID := mustTraceID("00112233445566778899aabbccddeeff")
	spanID := mustSpanID("0011223344556677")
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
		Remote:  true,
	}))

	if got := TraceIDFromContext(ctx); got != traceID.String() {
		t.Fatalf("TraceIDFromContext() = %q, want %q", got, traceID.String())
	}

	fallbackCtx := context.WithValue(context.Background(), ctxKey{}, "legacy-trace-id")
	if got := TraceIDFromContext(fallbackCtx); got != "legacy-trace-id" {
		t.Fatalf("TraceIDFromContext() fallback = %q, want %q", got, "legacy-trace-id")
	}
}

func TestInjectAndExtractTraceContext(t *testing.T) {
	withTestPropagator(t)

	ctx := newTraceContext(t)
	headers := InjectTraceContext(ctx, []pkgkafka.Header{
		{Key: "event", Value: []byte("audit.log.requested")},
	})

	if got := pkgkafka.HeaderValue(headers, "traceparent"); got == "" {
		t.Fatal("InjectTraceContext() did not write traceparent header")
	}
	if got := pkgkafka.HeaderValue(headers, "baggage"); got == "" {
		t.Fatal("InjectTraceContext() did not write baggage header")
	}

	extracted := ExtractTraceContext(context.Background(), headers)
	if got := TraceIDFromContext(extracted); got != "00112233445566778899aabbccddeeff" {
		t.Fatalf("TraceIDFromContext(extracted) = %q", got)
	}

	bag := baggage.FromContext(extracted)
	member := bag.Member("tenant_id")
	if member.Value() != "acme" {
		t.Fatalf("baggage tenant_id = %q, want %q", member.Value(), "acme")
	}
}

func TestTraceIDFromHeaders(t *testing.T) {
	withTestPropagator(t)

	t.Run("prefers explicit trace_id header", func(t *testing.T) {
		ctx := newTraceContext(t)
		headers := InjectTraceContext(ctx, nil)
		headers = append(headers, pkgkafka.Header{Key: traceIDHeader, Value: []byte("legacy-trace-id")})

		if got := TraceIDFromHeaders(headers); got != "legacy-trace-id" {
			t.Fatalf("TraceIDFromHeaders() = %q, want %q", got, "legacy-trace-id")
		}
	})

	t.Run("falls back to traceparent when trace_id is absent", func(t *testing.T) {
		ctx := newTraceContext(t)
		headers := InjectTraceContext(ctx, nil)

		if got := TraceIDFromHeaders(headers); got != "00112233445566778899aabbccddeeff" {
			t.Fatalf("TraceIDFromHeaders() = %q", got)
		}
	})
}

func TestContextWithSerializedTraceContext(t *testing.T) {
	withTestPropagator(t)

	t.Run("round trips trace context and baggage", func(t *testing.T) {
		ctx := newTraceContext(t)
		serialized := SerializeTraceContext(ctx)
		if serialized == "" {
			t.Fatal("SerializeTraceContext() returned empty string")
		}

		restored := ContextWithSerializedTraceContext(context.Background(), serialized)
		if got := TraceIDFromContext(restored); got != "00112233445566778899aabbccddeeff" {
			t.Fatalf("TraceIDFromContext(restored) = %q", got)
		}

		bag := baggage.FromContext(restored)
		member := bag.Member("tenant_id")
		if member.Value() != "acme" {
			t.Fatalf("restored baggage tenant_id = %q, want %q", member.Value(), "acme")
		}
	})

	t.Run("returns original context when payload is invalid", func(t *testing.T) {
		ctx := context.Background()
		restored := ContextWithSerializedTraceContext(ctx, "{bad json")

		if restored != ctx {
			t.Fatal("ContextWithSerializedTraceContext() should return original context on invalid payload")
		}
		if got := TraceIDFromContext(restored); got != "" {
			t.Fatalf("TraceIDFromContext(restored) = %q, want empty", got)
		}
	})
}

func TestSerializeHeadersTraceContext(t *testing.T) {
	withTestPropagator(t)

	ctx := newTraceContext(t)
	headers := InjectTraceContext(ctx, []pkgkafka.Header{
		{Key: "TraceParent", Value: []byte(pkgkafka.HeaderValue(InjectTraceContext(ctx, nil), "traceparent"))},
		{Key: "Baggage", Value: []byte(pkgkafka.HeaderValue(InjectTraceContext(ctx, nil), "baggage"))},
	})

	serialized := SerializeHeadersTraceContext(headers)
	if serialized == "" {
		t.Fatal("SerializeHeadersTraceContext() returned empty string")
	}

	restored := ContextWithSerializedTraceContext(context.Background(), serialized)
	if got := TraceIDFromContext(restored); got != "00112233445566778899aabbccddeeff" {
		t.Fatalf("TraceIDFromContext(restored) = %q", got)
	}
}

func withTestPropagator(t *testing.T) {
	t.Helper()

	previous := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	t.Cleanup(func() {
		otel.SetTextMapPropagator(previous)
	})
}

func newTraceContext(t *testing.T) context.Context {
	t.Helper()

	traceID := mustTraceID("00112233445566778899aabbccddeeff")
	spanID := mustSpanID("0011223344556677")
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
		Remote:  true,
	}))

	member, err := baggage.NewMember("tenant_id", "acme")
	if err != nil {
		t.Fatalf("baggage.NewMember() error = %v", err)
	}
	bag, err := baggage.New(member)
	if err != nil {
		t.Fatalf("baggage.New() error = %v", err)
	}

	return baggage.ContextWithBaggage(ctx, bag)
}

func mustTraceID(value string) trace.TraceID {
	traceID, err := trace.TraceIDFromHex(value)
	if err != nil {
		panic(err)
	}
	return traceID
}

func mustSpanID(value string) trace.SpanID {
	spanID, err := trace.SpanIDFromHex(value)
	if err != nil {
		panic(err)
	}
	return spanID
}
