package observability

import (
	"context"
	"fmt"
	"github.com/CIPFZ/video-cms/internal/core/config"
	"github.com/CIPFZ/video-cms/pkg/utils"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// Package observability -----------------------------
// @file        : metrics.go
// @author      : CIPFZ
// @time        : 2025/9/19 17:49
// @description :
// -------------------------------------------

// InitMetrics 初始化指标器
func InitMetrics(cfg config.Observability) (utils.ShutdownFunc, error) {
	if cfg.Exporter == "none" {
		return func(context.Context) error { return nil }, nil
	}

	// 1. 创建 Exporter
	exporter, err := createMetricsExporter(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics exporter: %w", err)
	}

	// 2. 创建 Resource (重用)
	res, err := createResource(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// 3. 创建 MeterProvider
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)), // Otel 推荐使用 PeriodicReader
		sdkmetric.WithResource(res),
	)

	// 4. ✨ 关键：设置为全局 MeterProvider
	otel.SetMeterProvider(mp)

	// 5. 返回 ShutdownFunc
	shutdown := func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := mp.Shutdown(ctx); err != nil {
			log.Printf("failed to shutdown metrics provider: %v", err)
			return err
		}
		return nil
	}

	return shutdown, nil
}

// createMetricsExporter (私有)
func createMetricsExporter(cfg config.Observability) (sdkmetric.Exporter, error) {
	switch cfg.Exporter {
	case "stdout":
		// stdoutmetric 默认会按时(Periodic)打印到控制台
		return stdoutmetric.New(stdoutmetric.WithPrettyPrint())
	case "otel":
		return createOtlpMetricExporter(context.Background(), cfg.OtelExporter)
	default:
		return nil, fmt.Errorf("unsupported metrics exporter: %s", cfg.Exporter)
	}
}

// createOtlpMetricExporter (私有)
func createOtlpMetricExporter(ctx context.Context, cfg config.OtelExporterConfig) (sdkmetric.Exporter, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("otel.endpoint is required")
	}

	var exp sdkmetric.Exporter
	var err error

	switch cfg.Protocol {
	case "grpc":
		opts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(cfg.Endpoint)}
		if cfg.Insecure {
			opts = append(opts, otlpmetricgrpc.WithTLSCredentials(insecure.NewCredentials()))
		}
		exp, err = otlpmetricgrpc.New(ctx, opts...)
	case "http":
		opts := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpoint(cfg.Endpoint)}
		if cfg.Insecure {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		exp, err = otlpmetrichttp.New(ctx, opts...)
	default:
		return nil, fmt.Errorf("unsupported otel.protocol: %s", cfg.Protocol)
	}
	return exp, err
}
