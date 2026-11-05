package migrations

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/CIPFZ/video-cms/internal/core/config"
	"github.com/CIPFZ/video-cms/internal/modules/media"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func editionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "cms.db")), &gorm.Config{})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Close() })
	return db
}

func TestMediaEditionCatalogAndGrants(t *testing.T) {
	db := editionTestDB(t)
	t.Setenv("SEED_ADMIN_ENABLED", "true")
	t.Setenv("SEED_ADMIN_PASSWORD", "migration-password")
	cfg := &config.Config{System: config.System{Environment: "dev", RouterPrefix: "/custom"}}
	migrate := func() { require.NoError(t, Run(context.Background(), db, cfg, zap.NewNop())) }
	migrate()
	require.NoError(t, Check(db))
	require.True(t, db.Migrator().HasTable(&media.CollectionSchedule{}))
	require.True(t, db.Migrator().HasTable(&media.PlayEpisode{}))
	require.False(t, db.Migrator().HasTable("proxy_instances"))
	var count int64
	require.NoError(t, db.Model(&model.SysMenu{}).Where("path LIKE ? OR path LIKE ?", "/magnet/%", "/proxy/%").Count(&count).Error)
	require.Zero(t, count)
	require.NoError(t, db.Model(&model.SysApi{}).Where("api_group IN ?", []string{"proxy", "magnet"}).Count(&count).Error)
	require.Zero(t, count)
	var library model.SysMenu
	require.NoError(t, db.Where("path = ?", "/media/library").First(&library).Error)
	require.NoError(t, db.Where("menu_id = ? AND authority_id = ?", library.ID, 1).First(&model.SysAuthorityMenu{}).Error)
	var stream model.SysApi
	require.NoError(t, db.Where("path = ? AND method = ?", "/custom/media/episodes/:id/stream", "GET").First(&stream).Error)
	require.NoError(t, db.Where("api_id = ? AND authority_id = ?", stream.ID, 1).First(&model.SysAuthorityApi{}).Error)
	rule := db.Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", "p", "1", stream.Path, "GET")
	require.NoError(t, rule.First(&model.SysCasbinRule{}).Error)

	// A subsequent installation run must preserve customized menus and revoked grants.
	require.NoError(t, db.Model(&library).Update("name", "My movies").Error)
	require.NoError(t, db.Where("menu_id = ? AND authority_id = ?", library.ID, 1).Delete(&model.SysAuthorityMenu{}).Error)
	migrate()
	require.NoError(t, db.First(&library, library.ID).Error)
	require.Equal(t, "My movies", library.Name)
	require.NoError(t, db.Model(&model.SysAuthorityMenu{}).Where("menu_id = ? AND authority_id = ?", library.ID, 1).Count(&count).Error)
	require.Zero(t, count)
}

func TestMediaEditionRejectsCombinedDatabaseBeforeMigration(t *testing.T) {
	for _, fixture := range []string{"checkpoint", "partial-table"} {
		t.Run(fixture, func(t *testing.T) {
			db := editionTestDB(t)
			if fixture == "checkpoint" {
				require.NoError(t, db.AutoMigrate(&schemaMigration{}))
				require.NoError(t, db.Create(&schemaMigration{Name: "20260924_media_collection_v9"}).Error)
			} else {
				require.NoError(t, db.Exec("CREATE TABLE proxy_instances (id INTEGER PRIMARY KEY)").Error)
				require.NoError(t, db.Exec("INSERT INTO proxy_instances VALUES (42)").Error)
			}
			err := Run(context.Background(), db, &config.Config{}, zap.NewNop())
			require.ErrorContains(t, err, "separate database")
			require.False(t, db.Migrator().HasTable("sys_migration_lock"))
			require.False(t, db.Migrator().HasTable(&media.Item{}))
			var count int64
			if fixture == "checkpoint" {
				require.NoError(t, db.Model(&schemaMigration{}).Count(&count).Error)
				require.EqualValues(t, 1, count)
				require.NoError(t, db.Where("name = ?", "20260924_media_collection_v9").First(&schemaMigration{}).Error)
			} else {
				require.NoError(t, db.Table("proxy_instances").Where("id = ?", 42).Count(&count).Error)
				require.EqualValues(t, 1, count)
			}
		})
	}
}

func TestMediaRunItemsUpgradeKeepsGrantsAndEditedSchedules(t *testing.T) {
	db := editionTestDB(t)
	t.Setenv("SEED_ADMIN_ENABLED", "true")
	t.Setenv("SEED_ADMIN_PASSWORD", "migration-password")
	cfg := &config.Config{System: config.System{Environment: "dev", RouterPrefix: "/custom"}}
	require.NoError(t, Run(context.Background(), db, cfg, zap.NewNop()))
	require.True(t, db.Migrator().HasTable(&media.CollectionRunItem{}))

	// Recreate an installation from before the run item step.
	var sources []media.Source
	require.NoError(t, db.Order("id").Find(&sources).Error)
	require.GreaterOrEqual(t, len(sources), 2)
	weekly := func(sourceID uint, enabled bool) media.CollectionSchedule {
		row := media.CollectionSchedule{SourceID: sourceID, Kind: "full", Cron: "30 3 * * 0", NextRunAt: time.Now()}
		require.NoError(t, db.Create(&row).Error)
		require.NoError(t, db.Model(&row).Update("enabled", enabled).Error)
		return row
	}
	untouched, edited := weekly(sources[0].ID, false), weekly(sources[1].ID, true)
	var listSources model.SysApi
	require.NoError(t, db.Where("path = ? AND method = ?", "/custom/media/sources", "GET").First(&listSources).Error)
	require.NoError(t, db.Where("api_id = ? AND authority_id = ?", listSources.ID, 1).Delete(&model.SysAuthorityApi{}).Error)
	require.NoError(t, db.Where("name IN ?", []string{mediaRunItemsVersion, Latest}).Delete(&schemaMigration{}).Error)
	require.NoError(t, db.Migrator().DropTable(&media.CollectionRunItem{}))
	require.NoError(t, db.Migrator().DropIndex(&media.Item{}, mediaItemsUpdatedIndex))

	require.NoError(t, Run(context.Background(), db, cfg, zap.NewNop()))
	require.NoError(t, Check(db))
	require.True(t, db.Migrator().HasTable(&media.CollectionRunItem{}))
	require.True(t, db.Migrator().HasIndex(&media.Item{}, mediaItemsUpdatedIndex))
	var items model.SysApi
	require.NoError(t, db.Where("path = ? AND method = ?", "/custom/media/runs/:id/items", "GET").First(&items).Error)
	require.NoError(t, db.Where("api_id = ? AND authority_id = ?", items.ID, 1).First(&model.SysAuthorityApi{}).Error)
	require.NoError(t, db.Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", "p", "1", items.Path, "GET").First(&model.SysCasbinRule{}).Error)
	var count int64
	require.NoError(t, db.Model(&model.SysAuthorityApi{}).Where("api_id = ? AND authority_id = ?", listSources.ID, 1).Count(&count).Error)
	require.Zero(t, count, "revoked grants stay revoked")

	require.NoError(t, db.First(&untouched, untouched.ID).Error)
	require.Equal(t, media.DefaultFullCron, untouched.Cron)
	require.True(t, untouched.NextRunAt.After(time.Now()))
	require.NoError(t, db.First(&edited, edited.ID).Error)
	require.Equal(t, "30 3 * * 0", edited.Cron)
}
