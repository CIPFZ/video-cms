package audit

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestSanitizeBody(t *testing.T) {
	for _, input := range []string{
		`{"password":"private-value","name":"kept","nested":[{"access_token":"private-value"}],"id":9007199254740993}`,
		`{"oldPassword":"private-value","new-password":"private-value","secretKey":"private-value"}`,
		`{"password":"private-value`,
		`password=private-value`,
		strings.Repeat(" ", MaxBodySize) + `{"password":"private-value"}`,
	} {
		output := SanitizeBody([]byte(input))
		require.NotContains(t, output, "private-value")
		require.LessOrEqual(t, len(output), MaxBodySize)
	}
	require.Contains(t, SanitizeBody([]byte(`{"id":9007199254740993,"name":"kept"}`)), "9007199254740993")
}

func TestRecorderDrainAndConcurrentClose(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	defer sqlDB.Close()
	require.NoError(t, db.AutoMigrate(&model.SysOperationLog{}))
	r := NewAuditRecorder(db, zap.NewNop())
	r.Push(model.SysOperationLog{Path: "/retained"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); r.Push(model.SysOperationLog{Path: "/concurrent"}); _ = r.Close(ctx) }()
	}
	wg.Wait()
	require.NoError(t, r.Close(ctx))
	r.Push(model.SysOperationLog{Path: "/after-close"})
	var count int64
	require.NoError(t, db.Model(&model.SysOperationLog{}).Where("path = ?", "/retained").Count(&count).Error)
	require.EqualValues(t, 1, count)
	require.NoError(t, db.Model(&model.SysOperationLog{}).Where("path = ?", "/after-close").Count(&count).Error)
	require.Zero(t, count)
}
