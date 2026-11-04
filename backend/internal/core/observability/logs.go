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
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// Package observability -----------------------------
// @file        : logs.go
// @author      : CIPFZ
// @time        : 2025/9/23 21:54
// @description :
// -------------------------------------------

// InitLogs 初始化日志推送 provider
func InitLogs(cfg config.Observability) (*sdklog.LoggerProvider, utils.ShutdownFunc, error) {
	// 1. 如果禁用了 Exporter，返回一个 "no-op" (空操作) shutdown
	if cfg.Exporter == "none" {
		return nil, func(context.Context) error { return nil }, nil
	}

	// 2. 创建 Exporter
	exporter, err := createLogsExporter(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create log exporter: %w", err)
	}

	// 3. 创建 Resource
	res, err := createResource(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// 4. 创建 LoggerProvider
	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(res),
	)

	// 5. ✨ 关键：设置为全局 LoggerProvider
	// 这允许 otel.GetLoggerProvider() 在您框架的任何地方工作
	global.SetLoggerProvider(lp)

	// 6. 返回一个封装了超时的 Graceful Shutdown 函数
	shutdown := func(ctx context.Context) error {
		// 为 shutdown 设置一个5秒的超时
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := lp.Shutdown(ctx); err != nil {
			otel.Handle(err) // Otel 推荐使用 Handle
			log.Printf("failed to shutdown log provider: %v", err)
			return err
		}
		return nil
	}

	return lp, shutdown, nil
}

// createLogsExporter (私有) 创建 Logs 导出器
func createLogsExporter(cfg config.Observability) (sdklog.Exporter, error) {
	switch cfg.Exporter {
	case "stdout":
		// 用于本地调试，打印到控制台
		return stdoutlog.New(stdoutlog.WithPrettyPrint())

	case "otel":
		// 厂商中立的 OTLP 导出器
		return createOtlpLogExporter(context.Background(), cfg.OtelExporter)

	case "none":
		return nil, nil // No-op

	default:
		return nil, fmt.Errorf("unsupported log exporter: %s", cfg.Exporter)
	}
}

// createOtlpLogExporter 变得厂商中立
func createOtlpLogExporter(ctx context.Context, cfg config.OtelExporterConfig) (sdklog.Exporter, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("otel.endpoint is required")
	}

	var exp sdklog.Exporter
	var err error

	switch cfg.Protocol {
	case "grpc":
		opts := []otlploggrpc.Option{
			otlploggrpc.WithEndpoint(cfg.Endpoint),
		}
		if cfg.Insecure {
			//opts = append(opts, otlploggrpc.WithInsecure())
			opts = append(opts, otlploggrpc.WithTLSCredentials(insecure.NewCredentials()))
		}
		exp, err = otlploggrpc.New(ctx, opts...)

	case "http":
		opts := []otlploghttp.Option{
			otlploghttp.WithEndpoint(cfg.Endpoint),
		}
		if cfg.Insecure {
			opts = append(opts, otlploghttp.WithInsecure())
		}
		exp, err = otlploghttp.New(ctx, opts...)

	default:
		return nil, fmt.Errorf("unsupported otel.protocol: %s", cfg.Protocol)
	}

	return exp, err
}
