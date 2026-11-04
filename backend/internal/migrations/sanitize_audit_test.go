package migrations

import (
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"testing"
)

func TestSanitizeAuditHistory(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	pool, _ := db.DB()
	pool.SetMaxOpenConns(1)
	defer pool.Close()
	require.NoError(t, db.AutoMigrate(&model.SysOperationLog{}))
	row := model.SysOperationLog{Body: `{"password":"legacy-secret","name":"kept"}`, Resp: `{"data":{"token":"legacy-secret"}}`}
	require.NoError(t, db.Create(&row).Error)
	require.NoError(t, sanitizeAuditHistory(db))
	require.NoError(t, sanitizeAuditHistory(db))
	require.NoError(t, db.First(&row, row.ID).Error)
	require.NotContains(t, row.Body, "legacy-secret")
	require.NotContains(t, row.Resp, "legacy-secret")
	require.Contains(t, row.Body, "kept")
	var count int64
	require.NoError(t, db.Model(&schemaMigration{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
}
