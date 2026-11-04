package seed

import (
	"path/filepath"
	"testing"

	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestFreshCMSSeedsOnlyAdminAndPreservesEditedMenuSettings(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "seed.db")), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, db.AutoMigrate(&model.SysUser{}, &model.SysAuthority{}, &model.SysMenu{}, &model.SysUserAuthority{}))
	opts := seedAdminOptions{Username: "admin", Password: "initial-test-password", AuthorityID: 1, DefaultRoute: "dashboard/workplace"}
	require.NoError(t, ensureAuthorities(db, opts))
	menuIDs, err := ensureBaseMenus(db)
	require.NoError(t, err)
	var rootPaths, systemPaths []string
	require.NoError(t, db.Model(&model.SysMenu{}).Where("parent_id = 0 AND hide_in_menu = ?", false).Order("sort").Pluck("path", &rootPaths).Error)
	require.Equal(t, []string{"/dashboard/workplace", "/sys", "/state", "/about"}, rootPaths)
	require.NoError(t, db.Model(&model.SysMenu{}).Where("parent_id = ?", menuIDs["sys_root"]).Order("sort").Pluck("path", &systemPaths).Error)
	require.Equal(t, []string{"/sys/user", "/sys/authority", "/sys/menu", "/sys/api", "/sys/api-token", "/sys/notice", "/sys/operation"}, systemPaths)
	require.NoError(t, ensureAdminUser(db, opts))
	var users []model.SysUser
	require.NoError(t, db.Find(&users).Error)
	require.Len(t, users, 1)
	require.Equal(t, "admin", users[0].Username)
	password := users[0].Password
	var menu model.SysMenu
	require.NoError(t, db.Where("path = ?", "/dashboard/workplace").First(&menu).Error)
	require.Equal(t, "工作台", menu.Name)
	require.Equal(t, "menu.dashboard.workplace", menu.Locale)
	require.NoError(t, db.Model(&menu).Updates(map[string]any{"name": "团队工作台", "sort": 5, "icon": "RocketOutlined"}).Error)
	opts.Password = "changed-seed-password"
	require.NoError(t, ensureAdminUser(db, opts))
	_, err = ensureBaseMenus(db)
	require.NoError(t, err)
	require.NoError(t, db.Find(&users).Error)
	require.Len(t, users, 1)
	require.Equal(t, password, users[0].Password)
	require.NoError(t, db.First(&menu, menu.ID).Error)
	require.Equal(t, "团队工作台", menu.Name)
	require.Equal(t, 5, menu.Sort)
	require.Equal(t, "RocketOutlined", menu.Icon)
}
