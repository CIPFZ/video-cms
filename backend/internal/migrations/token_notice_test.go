package migrations

import (
	"context"
	"github.com/CIPFZ/video-cms/internal/core/config"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"testing"
	"time"
)

func TestTokenNoticeUpgradePreservesHistoryAndDoesNotGrantTokens(t *testing.T) {
	database := cleanupTestDB(t)
	t.Setenv("SEED_ADMIN_ENABLED", "true")
	t.Setenv("SEED_ADMIN_PASSWORD", "migration-password")
	cfg := &config.Config{System: config.System{Environment: "dev", RouterPrefix: "/api/v1"}}
	require.NoError(t, Run(context.Background(), database, cfg, zap.NewNop()))
	notice := model.SysNotice{Title: "Existing notice", Content: "keep content", TargetType: model.NoticeTargetRoles}
	require.NoError(t, database.Create(&notice).Error)
	read := time.Now()
	receiver := model.SysNoticeReceiver{NoticeID: notice.ID, UserID: 1, ReadAt: &read}
	require.NoError(t, database.Create(&receiver).Error)
	token := model.SysApiToken{Name: "legacy", TokenHash: "unchanged-hash", Enabled: true}
	require.NoError(t, database.Create(&token).Error)
	// Simulate the pre-upgrade schema and missing catalog entries.
	require.NoError(t, database.Migrator().DropColumn(&model.SysNotice{}, "target_ids"))
	require.NoError(t, database.Where("name = ?", tokenNoticeVersion).Delete(&schemaMigration{}).Error)
	require.NoError(t, database.Unscoped().Where("path IN ?", []string{"/api/v1/open/token-info", "/api/v1/sys/api-token/options"}).Delete(&model.SysApi{}).Error)
	require.NoError(t, Run(context.Background(), database, cfg, zap.NewNop()))
	require.NoError(t, Run(context.Background(), database, cfg, zap.NewNop()))
	var restored model.SysNotice
	require.NoError(t, database.First(&restored, notice.ID).Error)
	require.Equal(t, notice.Content, restored.Content)
	require.Empty(t, restored.TargetIDs)
	var received model.SysNoticeReceiver
	require.NoError(t, database.First(&received, receiver.ID).Error)
	require.NotNil(t, received.ReadAt)
	require.True(t, read.Equal(*received.ReadAt))
	var restoredToken model.SysApiToken
	require.NoError(t, database.Preload("Apis").First(&restoredToken, token.ID).Error)
	require.Equal(t, token.TokenHash, restoredToken.TokenHash)
	require.Empty(t, restoredToken.Apis)
	var count int64
	require.NoError(t, database.Model(&model.SysApi{}).Where("path = ?", "/api/v1/open/token-info").Count(&count).Error)
	require.EqualValues(t, 1, count)
}
