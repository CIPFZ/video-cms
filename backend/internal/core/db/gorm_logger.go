package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	logger "github.com/CIPFZ/video-cms/internal/core/log"
	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type ZapGormLogger struct {
	ZapLogger                 *zap.Logger
	LogLevel                  gormlogger.LogLevel
	SlowThreshold             time.Duration
	IgnoreRecordNotFoundError bool
}

type gormLogConfig interface {
	LogLevel() gormlogger.LogLevel
}

func NewZapGormLogger(zapLogger *zap.Logger, cfg gormLogConfig) *ZapGormLogger {
	return &ZapGormLogger{
		ZapLogger:                 zapLogger,
		LogLevel:                  cfg.LogLevel(),
		SlowThreshold:             200 * time.Millisecond,
		IgnoreRecordNotFoundError: true,
	}
}

func (l *ZapGormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

func (l *ZapGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		logger.GetLogger(ctx).Info(fmt.Sprintf(msg, data...), zap.String("module", "gorm"))
	}
}

func (l *ZapGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		logger.GetLogger(ctx).Warn(fmt.Sprintf(msg, data...), zap.String("module", "gorm"))
	}
}

func (l *ZapGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		logger.GetLogger(ctx).Error(fmt.Sprintf(msg, data...), zap.String("module", "gorm"))
	}
}

func (l *ZapGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	log := logger.GetLogger(ctx).With(
		zap.String("module", "gorm"),
		zap.Duration("duration", elapsed),
	)

	sql, rows := fc()
	if rows != -1 {
		log = log.With(zap.Int64("rows", rows))
	}

	if err != nil && (!l.IgnoreRecordNotFoundError || !errors.Is(err, gorm.ErrRecordNotFound)) {
		log.Error("sql_error", zap.String("sql", sql), zap.Error(err))
		return
	}

	if l.SlowThreshold != 0 && elapsed > l.SlowThreshold {
		log.Warn("sql_slow", zap.String("sql", sql))
		return
	}

	if l.LogLevel == gormlogger.Info {
		log.Info("sql_exec", zap.String("sql", sql))
	}
}

// ParamsFilter keeps SQL placeholders in logs, including slow/error queries.
func (l *ZapGormLogger) ParamsFilter(_ context.Context, sql string, _ ...interface{}) (string, []interface{}) {
	return sql, nil
}
