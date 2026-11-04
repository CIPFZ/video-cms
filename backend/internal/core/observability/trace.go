package observability

import (
	"context"
	"fmt"
	"github.com/CIPFZ/video-cms/pkg/utils"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"time"

	"github.com/CIPFZ/video-cms/internal/core/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Package observability -----------------------------
// @file        : trace.go
// @author      : CIPFZ
// @time        : 2025/9/19 18:27
// @description :
// -------------------------------------------

// InitTraces 初始化 Tracer Provider
func InitTraces(cfg config.Observability) (utils.ShutdownFunc, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	if cfg.Exporter == "none" {
		return func(context.Context) error { return nil }, nil
	}

	// 1. 创建 Exporter
	exporter, err := createTracesExporter(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// 2. 创建 Resource (重用)
	res, err := createResource(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// 3. ✨ 关键：创建采样器 (Sampler)
	// 这是企业级推荐的采样器：
	// - 如果父 Span (来自上游服务) 被采样了，则本服务也采样。
	// - 如果是根 Span (第一个 Span)，则根据我们配置的 ratio 决定是否采样。
	sampler := sdktrace.ParentBased(
		sdktrace.TraceIDRatioBased(cfg.TraceSampleRatio),
	)

	// 4. 创建 TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// 5. ✨ 关键：设置为全局 TracerProvider
	// 这是正确的全局设置函数
	otel.SetTracerProvider(tp)

	// 6. 返回 ShutdownFunc
	shutdown := func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			log.Printf("failed to shutdown trace provider: %v", err)
			return err
		}
		return nil
	}

	return shutdown, nil
}

// createTracesExporter (私有)
func createTracesExporter(cfg config.Observability) (sdktrace.SpanExporter, error) {
	switch cfg.Exporter {
	case "stdout":
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	case "otel":
		return createOtlpTraceExporter(context.Background(), cfg.OtelExporter)
	default:
		return nil, fmt.Errorf("unsupported trace exporter: %s", cfg.Exporter)
	}
}

// createOtlpTraceExporter (私有)
func createOtlpTraceExporter(ctx context.Context, cfg config.OtelExporterConfig) (sdktrace.SpanExporter, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("otel.endpoint is required")
	}

	var exp sdktrace.SpanExporter
	var err error

	switch cfg.Protocol {
	case "grpc":
		opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(cfg.Endpoint)}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithTLSCredentials(insecure.NewCredentials()))
		}
		exp, err = otlptracegrpc.New(ctx, opts...)
	case "http":
		opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(cfg.Endpoint)}
		if cfg.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		exp, err = otlptracehttp.New(ctx, opts...)
	default:
		return nil, fmt.Errorf("unsupported otel.protocol: %s", cfg.Protocol)
	}
	return exp, err
}
