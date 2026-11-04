package db

import (
	"context"
	"fmt"
	"time"

	"github.com/CIPFZ/video-cms/internal/core/config"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func InitPostgres(p config.Postgres, logger *zap.Logger) (*gorm.DB, error) {
	if p.Dbname == "" {
		return nil, fmt.Errorf("postgres dbname is empty")
	}

	gormDB, err := gorm.Open(postgres.Open(p.Dsn()), &gorm.Config{
		Logger: NewZapGormLogger(logger, p),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: p.Singular,
		},
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres: %w", err)
	}

	if err = gormDB.Use(otelgorm.NewPlugin(otelgorm.WithoutQueryVariables(), otelgorm.WithDBName(p.Dbname))); err != nil {
		return nil, fmt.Errorf("failed to use otelgorm plugin: %w", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}
	sqlDB.SetMaxIdleConns(p.MaxIdleConns)
	sqlDB.SetMaxOpenConns(p.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return gormDB, nil
}
