package logger

import (
	"context"
	"github.com/CIPFZ/video-cms/pkg/utils"
	"os"
	"path/filepath"
	"time"

	"github.com/CIPFZ/video-cms/internal/core/config"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Package logger -----------------------------
// @file        : logger.go
// @author      : CIPFZ
// @time        : 2025/9/19 17:24
// @description :
// -------------------------------------------

// NewLogger 根据配置创建 zap.Logger
func NewLogger(cfg *config.Logger, otelCfg *config.OTELLoggerConfig) (*zap.Logger, error) {
	// 解析日志级别
	var lvl zapcore.Level
	if err := lvl.UnmarshalText([]byte(cfg.Level)); err != nil {
		lvl = zapcore.InfoLevel
	}

	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stack",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeDuration: zapcore.NanosDurationEncoder,
		LineEnding:     zapcore.DefaultLineEnding,
	}

	// 支持 console | json
	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	}

	// outputs
	var cores []zapcore.Core

	// helper to create writer for file using lumberjack
	if cfg.Output == "file" || cfg.Output == "both" {
		if cfg.FilePath == "" {
			cfg.FilePath = "logs/app.log"
		}
		_ = os.MkdirAll(filepath.Dir(cfg.FilePath), 0755)

		if cfg.MaxSizeMB == 0 {
			cfg.MaxSizeMB = 100
		}
		if cfg.MaxBackups == 0 {
			cfg.MaxBackups = 7
		}
		if cfg.MaxAgeDays == 0 {
			cfg.MaxAgeDays = 30
		}
		lumber := &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSizeMB, // megabytes
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAgeDays, // days
			Compress:   cfg.Compress,
		}
		writer := zapcore.AddSync(lumber)
		core := zapcore.NewCore(encoder, writer, lvl)
		cores = append(cores, core)
	}

	// stdout
	if cfg.Output == "stdout" || cfg.Output == "both" {
		stdoutWriter := zapcore.Lock(os.Stdout)
		core := zapcore.NewCore(encoder, stdoutWriter, lvl)
		cores = append(cores, core)
	}

	// fallback if no core configured
	if len(cores) == 0 {
		// default to stdout
		stdoutWriter := zapcore.Lock(os.Stdout)
		core := zapcore.NewCore(encoder, stdoutWriter, lvl)
		cores = append(cores, core)
	}

	// 如果 ooProvider 不为 nil，就追加一个 core
	if otelCfg != nil && otelCfg.LogProvider != nil {
		ooCore := NewOTELLogCore(otelCfg, lvl)
		cores = append(cores, ooCore)
	}

	core := zapcore.NewTee(cores...)

	// optional sampling
	if cfg.EnableSample {
		core = zapcore.NewSamplerWithOptions(core, time.Second, 100, 100)
	}

	return zap.New(core, zap.AddCaller()), nil
}

// Sync flushes any buffered logs (调用时请忽略返回错误)
func Sync(l *zap.Logger) {
	_ = l.Sync()
}

// GetLogger 支持从 gin.Context 或 context.Context 中获取
func GetLogger(ctx interface{}) *zap.Logger {
	var logger interface{}

	switch c := ctx.(type) {
	case *gin.Context:
		// 优先从 Gin Keys 取
		if v, exists := c.Get(utils.LoggerKey); exists {
			logger = v
		} else {
			// 兜底：如果 Gin Keys 没有，尝试从 Request.Context 取
			logger = c.Request.Context().Value(utils.LoggerKey)
		}
	case context.Context:
		// 从标准 Context 取
		logger = c.Value(utils.LoggerKey)
	}

	// 只要 logger 不为 nil 且类型断言成功，就返回
	if logger != nil {
		if l, ok := logger.(*zap.Logger); ok {
			return l
		}
	}
	return zap.L()
}

// Use the upstream bridge for typed fields, trace context and evolving OTel APIs.
func NewOTELLogCore(cfg *config.OTELLoggerConfig, lvl zapcore.LevelEnabler) zapcore.Core {
	core := otelzap.NewCore("base-frame", otelzap.WithLoggerProvider(cfg.LogProvider))
	return &levelCore{Core: core, level: lvl}
}

type levelCore struct {
	zapcore.Core
	level zapcore.LevelEnabler
}

func (c *levelCore) Enabled(level zapcore.Level) bool {
	return c.level.Enabled(level) && c.Core.Enabled(level)
}
func (c *levelCore) With(fields []zapcore.Field) zapcore.Core {
	return &levelCore{Core: c.Core.With(fields), level: c.level}
}
func (c *levelCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}
	return checked
}
