package db

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"time"

	"github.com/CIPFZ/video-cms/internal/core/config"

	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// InitMysql 通过传入配置初始化Mysql数据库
func InitMysql(m config.MySQL, logger *zap.Logger) (*gorm.DB, error) {
	if m.Dbname == "" {
		// 企业级：返回 error 而不是 panic
		return nil, fmt.Errorf("mysql dbname is empty")
	}

	mysqlConfig := mysql.Config{
		DSN:                       m.Dsn(),
		DefaultStringSize:         191,
		SkipInitializeWithVersion: false,
	}

	// ✨ 关键：使用我们自定义的 ZapGormLogger
	zapGormLogger := NewZapGormLogger(logger, m)

	gormCfg := &gorm.Config{
		Logger: zapGormLogger, // 替换原来的 logger.New(...)
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   m.Prefix,
			SingularTable: m.Singular,
		},
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	var db *gorm.DB
	var err error

	// 1. 打开连接
	db, err = gorm.Open(mysql.New(mysqlConfig), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql: %w", err)
	}

	// 2. 注册 Otel 插件 (用于生成 Trace Span)
	if err = db.Use(otelgorm.NewPlugin(otelgorm.WithoutQueryVariables(),
		otelgorm.WithDBName(m.Dbname),
	)); err != nil {
		return nil, fmt.Errorf("failed to use otelgorm plugin: %w", err)
	}

	// 3. 连接池设置
	db.InstanceSet("gorm:table_options", "ENGINE="+m.Engine)
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(m.MaxIdleConns)
	sqlDB.SetMaxOpenConns(m.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 4. 健康检查 (Ping)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping mysql: %w", err)
	}

	return db, nil
}
