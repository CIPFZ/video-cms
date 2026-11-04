package bootstrap

import (
	"context"
	"fmt"
	"github.com/CIPFZ/video-cms/internal/core/session"
	"github.com/CIPFZ/video-cms/internal/migrations"
	"github.com/CIPFZ/video-cms/internal/modules/media"
	"time"

	"github.com/CIPFZ/video-cms/internal/core/claims"
	"github.com/CIPFZ/video-cms/internal/core/db"
	"github.com/CIPFZ/video-cms/internal/core/file"
	"github.com/CIPFZ/video-cms/internal/core/jwt"
	"github.com/CIPFZ/video-cms/pkg/utils"
	"go.opentelemetry.io/otel"

	"github.com/CIPFZ/video-cms/internal/core/audit"
	"github.com/CIPFZ/video-cms/internal/core/config"
	"github.com/CIPFZ/video-cms/internal/core/i18n"
	corelog "github.com/CIPFZ/video-cms/internal/core/log"
	"github.com/CIPFZ/video-cms/internal/core/observability"
	"github.com/CIPFZ/video-cms/internal/svc"

	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.uber.org/zap"
)

// initializeSystem 初始化核心组件并组装
func Initialize(path string, serviceCtx *svc.ServiceContext) (shutdowns []utils.ShutdownFunc, initErr error) {
	defer func() {
		if initErr != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			for i := len(shutdowns) - 1; i >= 0; i-- {
				_ = shutdowns[i](ctx)
			}
		}
	}()
	var lp *sdklog.LoggerProvider
	var logShutdown utils.ShutdownFunc
	var err error

	// Step 1: 加载配置 (使用 core/config)
	var v *viper.Viper
	cfg, v, err := config.Load(path)
	if err != nil {
		return shutdowns, fmt.Errorf("config load failed: %w", err)
	}
	if err := config.ValidateServer(cfg); err != nil {
		return shutdowns, err
	}
	serviceCtx.Config = cfg
	serviceCtx.Viper = v

	// Step 2: 初始化 Otel (core/trace)
	{
		// Logs
		lp, logShutdown, err = observability.InitLogs(cfg.Observable)
		if err != nil {
			return shutdowns, fmt.Errorf("otel logs init failed: %w", err)
		}

		shutdowns = append(shutdowns, logShutdown)

		// Traces
		traceShutdown, err := observability.InitTraces(cfg.Observable)
		if err != nil {
			return shutdowns, fmt.Errorf("otel traces init failed: %w", err)
		}
		shutdowns = append(shutdowns, traceShutdown)

		// Metrics
		metricShutdown, err := observability.InitMetrics(cfg.Observable)
		if err != nil {
			return shutdowns, fmt.Errorf("otel metrics init failed: %w", err)
		}
		shutdowns = append(shutdowns, metricShutdown)

		// Runtime Metrics
		fmt.Println("正在启动 Go Runtime 指标收集...")
		if err = runtime.Start(
			runtime.WithMeterProvider(otel.GetMeterProvider()),
			runtime.WithMinimumReadMemStatsInterval(15*time.Second),
		); err != nil {
			return shutdowns, fmt.Errorf("runtime metrics start failed: %w", err)
		}
	}

	// Step 3: 初始化 Logger (core/log)
	serviceCtx.Logger, err = corelog.NewLogger(&cfg.Logger, &config.OTELLoggerConfig{
		LogProvider: lp, // 传入 SDK Provider
	})
	if err != nil {
		return shutdowns, fmt.Errorf("logger init failed: %w", err)
	}
	// 替换全局 Logger，方便 middleware 使用 zap.L()
	zap.ReplaceGlobals(serviceCtx.Logger)
	// 修正: global.SetLoggerProvider 已经在 InitLogs 里做了，这里不需要再做

	serviceCtx.Logger.Info("✅ 基础设施初始化：配置、可观测性、日志")

	// Step 4: 国际化 (core/i18n)
	serviceCtx.I18n, err = i18n.NewI18n(cfg.I18n, serviceCtx.Logger)
	if err != nil {
		return shutdowns, fmt.Errorf("i18n init failed: %w", err)
	}

	// Step 5: 数据库连接 (core/db)
	// MySQL (GORM)
	serviceCtx.DB, err = db.InitDatabase(cfg.Database, serviceCtx.Logger)
	if err != nil {
		return shutdowns, fmt.Errorf("database init failed: %w", err)
	}
	sqlDB, err := serviceCtx.DB.DB()
	if err != nil {
		return shutdowns, fmt.Errorf("SQL pool: %w", err)
	}
	shutdowns = append(shutdowns, func(context.Context) error { return sqlDB.Close() })
	if err := migrations.Check(serviceCtx.DB); err != nil {
		return shutdowns, fmt.Errorf("schema check failed: %w", err)
	}

	// Redis
	if cfg.System.UseRedis {
		serviceCtx.Redis, err = db.InitRedis(cfg.Redis)
		if err != nil {
			return shutdowns, fmt.Errorf("redis init failed: %w", err)
		}
		shutdowns = append(shutdowns, func(ctx context.Context) error {
			if serviceCtx.Redis == nil {
				return nil
			}
			return serviceCtx.Redis.Close()
		})
	} else {
		serviceCtx.Logger.Warn("redis disabled; SQL sessions remain authoritative")
	}

	// Mongo
	if cfg.System.UseMongo {
		serviceCtx.Mongo, err = db.InitMongo(cfg.Mongo)
		if err != nil {
			return shutdowns, fmt.Errorf("mongo init failed: %w", err)
		}
		// 注册 Mongo 关停
		shutdowns = append(shutdowns, func(ctx context.Context) error {
			serviceCtx.Logger.Info("🛑 正在关闭 MongoDB...")
			return serviceCtx.Mongo.Close(ctx)
		})
	}

	// Step 6: 权限 Casbin
	serviceCtx.Policy, err = claims.NewPolicyManager(serviceCtx.DB)
	if err != nil {
		return shutdowns, fmt.Errorf("casbin init failed: %w", err)
	}
	serviceCtx.Sessions = session.NewStore(serviceCtx.DB)
	workerCtx, stopWorker := context.WithCancel(context.Background())
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-workerCtx.Done():
				return
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(workerCtx, 10*time.Second)
				if err := serviceCtx.Sessions.Cleanup(ctx); err != nil {
					serviceCtx.Logger.Warn("session cleanup failed", zap.Error(err))
				}
				cancel()
			}
		}
	}()
	shutdowns = append(shutdowns, func(ctx context.Context) error {
		stopWorker()
		select {
		case <-workerDone:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	serviceCtx.Logger.Info("Casbin 初始化完成")

	// Step 7: JWT (pkg/utils)
	// 依赖 Redis (ServiceContext已持有)
	serviceCtx.JWT = jwt.NewJWT(serviceCtx.Config.JWT, serviceCtx.Logger, serviceCtx.Redis)

	// Step 8: 审计日志 (core/audit)
	// ✨ 关键：使用 core 层的 AuditRecorder，解耦循环依赖
	auditRecorder := audit.NewAuditRecorder(serviceCtx.DB, serviceCtx.Logger)
	serviceCtx.AuditRecorder = auditRecorder
	shutdowns = append(shutdowns, auditRecorder.Close)

	// Step 9: 初始化 OSS
	serviceCtx.OSS = file.NewFileService(serviceCtx.Config.File, serviceCtx.Logger)

	serviceCtx.MediaCollector, err = media.NewService(media.Config{DB: serviceCtx.DB})
	if err != nil {
		return shutdowns, fmt.Errorf("media collector init: %w", err)
	}
	if err := serviceCtx.MediaCollector.Start(context.Background()); err != nil {
		return shutdowns, fmt.Errorf("media collector scheduler: %w", err)
	}
	shutdowns = append(shutdowns, func(ctx context.Context) error { return serviceCtx.MediaCollector.Close(ctx) })

	serviceCtx.Logger.Info("✅ 系统核心组件组装完成")
	return shutdowns, nil
}
