package migrations

import (
	"testing"

	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/stretchr/testify/require"
)

func TestNormalizeBaselineCleansTestDataAndPreservesRealAccounts(t *testing.T) {
	db := cleanupTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.SysNotice{}, &model.SysNoticeReceiver{}))
	for _, statement := range []string{
		"INSERT INTO sys_authorities (authority_id) VALUES (1), (888)",
		"INSERT INTO sys_users (id, username, password, authority_id) VALUES (1, 'admin', 'unchanged-admin-hash', 1), (2, 'e2e_user_b_123456', 'test', 888), (3, 'teammate', 'unchanged-user-hash', 888), (4, 'smoke_user_20260910', 'test', 888)",
		"INSERT INTO sys_user_authorities (user_id, authority_id) VALUES (1, 1), (2, 888), (3, 888), (4, 888)",
		"INSERT INTO sys_menus (id, name, locale, path) VALUES (1, 'menu.dashboard.workplace', 'menu.dashboard.workplace', '/dashboard/workplace'), (2, '自定义用户名称', 'menu.system.user', '/sys/user')",
		"INSERT INTO sys_notices (id, title, content, target_type, created_by) VALUES (1, 'E2E Admin Notice 123456', 'test', 'users', 1), (2, 'Team notice', 'keep', 'all', 1)",
		"INSERT INTO sys_notice_receivers (notice_id, user_id) VALUES (1, 2), (2, 1), (2, 2)",
		"INSERT INTO sys_operation_logs (user_id) VALUES (1), (2), (4)",
		"INSERT INTO sys_api_tokens (id, name, token_hash, created_by) VALUES (1, 'real', 'keep-hash', 1), (2, 'test', 'test-hash', 2)",
		"INSERT INTO sys_apis (id, path) VALUES (1, '/test')",
		"INSERT INTO sys_api_token_apis (api_token_id, api_id) VALUES (1, 1), (2, 1)",
	} {
		require.NoError(t, db.Exec(statement).Error)
	}
	require.NoError(t, normalizeCMSBaseline(db))
	for table, expected := range map[string]int64{
		"sys_users": 2, "sys_user_authorities": 2, "sys_notices": 1, "sys_notice_receivers": 1,
		"sys_operation_logs": 1, "sys_api_tokens": 1, "sys_api_token_apis": 1,
	} {
		var count int64
		require.NoError(t, db.Unscoped().Table(table).Count(&count).Error)
		require.Equal(t, expected, count, table)
	}
	var user model.SysUser
	require.NoError(t, db.First(&user, 1).Error)
	require.Equal(t, "unchanged-admin-hash", user.Password)
	var menus []model.SysMenu
	require.NoError(t, db.Order("id").Find(&menus).Error)
	require.Equal(t, "工作台", menus[0].Name)
	require.Equal(t, "menu.dashboard.workplace", menus[0].Locale)
	require.Equal(t, "自定义用户名称", menus[1].Name)
	// A completed migration is not a permanent single-user restriction.
	require.NoError(t, db.Exec("INSERT INTO sys_users (username, authority_id) VALUES ('e2e_user_987654', 888)").Error)
	require.NoError(t, normalizeCMSBaseline(db))
	var count int64
	require.NoError(t, db.Model(&model.SysUser{}).Count(&count).Error)
	require.Equal(t, int64(3), count)
}

func TestNormalizeBaselineOnFreshDatabase(t *testing.T) {
	db := cleanupTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.SysNotice{}, &model.SysNoticeReceiver{}))
	require.NoError(t, normalizeCMSBaseline(db))
	require.NoError(t, normalizeCMSBaseline(db))
}
