package migrations

import (
	"context"
	"testing"

	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/CIPFZ/video-cms/internal/modules/system/repository"
	"github.com/stretchr/testify/require"
)

func TestReorderCMSMenusPreservesMetadataAndLaterCustomization(t *testing.T) {
	db := cleanupTestDB(t)
	for _, statement := range []string{
		"INSERT INTO sys_menus (id, path, name, icon, sort, parent_id, hide_in_menu) VALUES (1, '/dashboard/workplace', '团队工作台', 'RocketOutlined', 1, 0, 0), (2, '/sys', '系统管理', 'SettingOutlined', 10, 0, 0), (3, '/state', '服务器状态', 'CloudServerOutlined', 2, 0, 0), (4, '/about', '关于', 'InfoCircleOutlined', 3, 0, 0), (5, '/account/settings', '个人设置', 'UserOutlined', 99, 0, 1), (6, '/sys/operation', '操作日志', 'HistoryOutlined', 6, 2, 0), (7, '/sys/notice', '通知公告', 'NotificationOutlined', 7, 2, 0), (8, '/sys/user', '用户管理', 'UserOutlined', 1, 2, 0), (9, '/bench', 'Bench', 'ExperimentOutlined', 35, 0, 0)",
		"INSERT INTO sys_authorities (authority_id) VALUES (1)",
		"INSERT INTO sys_authority_menus (authority_id, menu_id) SELECT 1, id FROM sys_menus",
	} {
		require.NoError(t, db.Exec(statement).Error)
	}
	require.NoError(t, reorderCMSMenus(db))
	repo := repository.NewMenuRepository(db)
	all, err := repo.GetAll(context.Background())
	require.NoError(t, err)
	authorized, err := repo.GetByAuthorityId(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, all, authorized, "management and navigation use the same ordering")
	var roots, children []string
	for _, menu := range all {
		if menu.ParentId == 0 && !menu.HideInMenu {
			roots = append(roots, menu.Path)
		} else if menu.ParentId == 2 {
			children = append(children, menu.Path)
		}
	}
	require.Equal(t, []string{"/dashboard/workplace", "/sys", "/bench", "/state", "/about"}, roots)
	require.Equal(t, []string{"/sys/user", "/sys/notice", "/sys/operation"}, children)
	var dashboard model.SysMenu
	require.NoError(t, db.First(&dashboard, 1).Error)
	require.Equal(t, "团队工作台", dashboard.Name)
	require.Equal(t, "RocketOutlined", dashboard.Icon)
	require.Equal(t, 10, dashboard.Sort)
	var hidden model.SysMenu
	require.NoError(t, db.First(&hidden, 5).Error)
	require.True(t, hidden.HideInMenu)
	require.Equal(t, 999, hidden.Sort)
	require.NoError(t, db.Model(&dashboard).Update("sort", 7).Error)
	require.NoError(t, reorderCMSMenus(db))
	require.NoError(t, db.First(&dashboard, 1).Error)
	require.Equal(t, 7, dashboard.Sort, "migration must not reset subsequent manual ordering")
}

func TestReorderCMSMenusOnFreshDatabase(t *testing.T) {
	db := cleanupTestDB(t)
	require.NoError(t, reorderCMSMenus(db))
	require.NoError(t, reorderCMSMenus(db))
	var count int64
	require.NoError(t, db.Model(&model.SysMenu{}).Count(&count).Error)
	require.Zero(t, count, "menu creation belongs to normal seeding")
}
