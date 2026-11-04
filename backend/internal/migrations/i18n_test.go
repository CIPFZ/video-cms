package migrations

import (
	"context"
	"github.com/CIPFZ/video-cms/internal/core/config"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/CIPFZ/video-cms/internal/modules/system/seed"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"testing"
	"time"
)

func TestI18nUpgradePreservesCustomMenus(t *testing.T) {
	db := cleanupTestDB(t)
	require.NoError(t, db.AutoMigrate(&schemaMigration{}))
	require.NoError(t, db.Create(&schemaMigration{Name: baselineVersion, AppliedAt: time.Now()}).Error)
	// Simulate the previous release: no English column, with existing grants and order.
	require.NoError(t, db.Exec("INSERT INTO sys_menus (id,name,locale,path,sort) VALUES (1,'工作台','menu.dashboard.workplace','/my-work',37),(2,'自定义页面','menu.system.user','/custom',99)").Error)
	if db.Migrator().HasColumn(&model.SysMenu{}, "name_en") {
		require.NoError(t, db.Migrator().DropColumn(&model.SysMenu{}, "name_en"))
	}
	require.Error(t, Check(db))
	require.NoError(t, Run(context.Background(), db, &config.Config{}, zap.NewNop()))
	require.NoError(t, Check(db))
	var menu, custom model.SysMenu
	require.NoError(t, db.First(&menu, 1).Error)
	require.Equal(t, seed.MenuNamesEn["menu.dashboard.workplace"], menu.NameEn)
	require.Equal(t, "/my-work", menu.Path)
	require.Equal(t, 37, menu.Sort)
	require.NoError(t, db.First(&custom, 2).Error)
	require.Empty(t, custom.NameEn)
	require.Equal(t, "自定义页面", custom.Name)
	require.NoError(t, db.Model(&menu).Update("name_en", "Our workspace").Error)
	require.NoError(t, Run(context.Background(), db, &config.Config{}, zap.NewNop()))
	require.NoError(t, db.First(&menu, 1).Error)
	require.Equal(t, "Our workspace", menu.NameEn)
}
