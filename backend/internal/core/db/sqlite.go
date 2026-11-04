package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/CIPFZ/video-cms/internal/core/config"
	"github.com/glebarez/sqlite"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func InitSQLite(s config.SQLite, logger *zap.Logger) (*gorm.DB, error) {
	if s.Path == "" {
		return nil, fmt.Errorf("sqlite path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create sqlite directory: %w", err)
	}

	gormDB, err := gorm.Open(sqlite.Open(s.Path), &gorm.Config{
		Logger: NewZapGormLogger(logger, s),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: s.Singular,
		},
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite3: %w", err)
	}

	if err = gormDB.Use(otelgorm.NewPlugin(otelgorm.WithoutQueryVariables(), otelgorm.WithDBName(s.DBName()))); err != nil {
		return nil, fmt.Errorf("failed to use otelgorm plugin: %w", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}
	sqlDB.SetMaxIdleConns(s.MaxIdleConns)
	sqlDB.SetMaxOpenConns(s.MaxOpenConns)

	if err := applySQLitePragmas(context.Background(), gormDB, s); err != nil {
		return nil, err
	}
	if err := sqlDB.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite3: %w", err)
	}
	return gormDB, nil
}

func applySQLitePragmas(ctx context.Context, db *gorm.DB, s config.SQLite) error {
	foreignKeys := "OFF"
	if s.ForeignKeys {
		foreignKeys = "ON"
	}
	if err := db.WithContext(ctx).Exec("PRAGMA foreign_keys = " + foreignKeys).Error; err != nil {
		return fmt.Errorf("failed to configure sqlite foreign keys: %w", err)
	}
	if s.WAL {
		if err := db.WithContext(ctx).Exec("PRAGMA journal_mode = WAL").Error; err != nil {
			return fmt.Errorf("failed to configure sqlite wal mode: %w", err)
		}
	}
	if err := db.WithContext(ctx).Exec(fmt.Sprintf("PRAGMA busy_timeout = %d", s.BusyTimeoutMS)).Error; err != nil {
		return fmt.Errorf("failed to configure sqlite busy timeout: %w", err)
	}
	return nil
}
