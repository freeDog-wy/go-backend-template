// Package tracing 负责 OpenTelemetry 链路追踪的初始化。
package tracing

import (
	"context"
	"fmt"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Init 初始化 OpenTelemetry TracerProvider。
//   - "development" → 控制台输出 trace（PrettyPrint），全采样
//   - "production"  → 控制台输出 trace（JSON），按比例采样
//
// 返回的 TracerProvider 已注册为全局。调用方需在程序退出前执行 Shutdown。
func Init(mode string) (*sdktrace.TracerProvider, error) {
	exporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
	)
	if err != nil {
		return nil, fmt.Errorf("create stdout exporter: %w", err)
	}

	sampler := sdktrace.AlwaysSample()
	if mode != "development" {
		sampler = sdktrace.TraceIDRatioBased(0.1) // 生产环境 10% 采样
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exporter),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, nil
}

// Shutdown 优雅关闭 TracerProvider，flush 所有未发送的 span。
func Shutdown(ctx context.Context, tp *sdktrace.TracerProvider) {
	if tp == nil {
		return
	}
	if err := tp.Shutdown(ctx); err != nil {
		log.Printf("[tracing] shutdown error: %v\n", err)
	}
}
