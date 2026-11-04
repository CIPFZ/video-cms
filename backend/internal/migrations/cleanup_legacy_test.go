package migrations

import (
	"path/filepath"
	"testing"

	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func cleanupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "migration.db")), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, db.AutoMigrate(&model.SysUser{}, &model.SysAuthority{}, &model.SysMenu{},
		&model.SysApi{}, &model.SysAuthorityMenu{}, &model.SysAuthorityApi{}, &model.SysUserAuthority{},
		&model.SysApiToken{}, &model.SysApiTokenApi{}, &model.SysCasbinRule{}, &model.SysOperationLog{}))
	return db
}

func TestCleanupLegacyModulesPreservesCMSAndCanRetry(t *testing.T) {
	db := cleanupTestDB(t)
	for _, table := range legacyModuleTables {
		require.NoError(t, db.Exec("CREATE TABLE "+table+" (id INTEGER PRIMARY KEY)").Error)
		require.NoError(t, db.Exec("INSERT INTO "+table+" (id) VALUES (1)").Error)
	}
	for _, statement := range []string{
		"INSERT INTO sys_authorities (authority_id, authority_name, default_router, parent_id) VALUES (1, 'Administrator', 'dashboard/workplace', 0), (10010, 'PluginProvider', 'plugin/project-management', 0), (10013, 'PluginReviewer', 'plugin/work-order-pool', 0), (42, 'Team', '/poetry/poem', 10010)",
		"INSERT INTO sys_users (id, username, password, authority_id) VALUES (1, 'admin', 'keep-admin-hash', 1), (2, 'former-provider', 'keep-user-hash', 10010)",
		"INSERT INTO sys_user_authorities (user_id, authority_id) VALUES (1, 1), (2, 10010), (2, 42)",
		"INSERT INTO sys_menus (id, path, component, parent_id) VALUES (1, '/sys', 'components/RouterLayout', 0), (2, '/plugin', 'components/RouterLayout', 0), (3, '/renamed-child', 'custom', 2), (4, '/poetry/poem', 'poetry/poem', 0), (5, '/sys/plugin-master', 'sys/plugin-master', 1), (6, '/sys/api-token', 'sys/api-token', 1)",
		"INSERT INTO sys_authority_menus (authority_id, menu_id) VALUES (1, 1), (1, 2), (1, 3), (1, 4), (1, 5), (10010, 1), (1, 6)",
		"INSERT INTO sys_apis (id, path, method) VALUES (1, '/custom/sys/menu/getMenu', 'GET'), (2, '/custom/plugin/release/claim', 'POST'), (3, '/api/v1/poetry/poem/list', 'GET')",
		"INSERT INTO sys_authority_apis (authority_id, api_id) VALUES (1, 1), (1, 2), (1, 3), (10013, 1)",
		"INSERT INTO sys_api_tokens (id, name, token_hash) VALUES (1, 'automation', 'keep-token-hash')",
		"INSERT INTO sys_api_token_apis (api_token_id, api_id) VALUES (1, 1), (1, 2), (1, 3)",
		"INSERT INTO sys_casbin_rules (ptype, v0, v1, v2) VALUES ('p', '1', '/custom/sys/menu/getMenu', 'GET'), ('p', '1', '/custom/plugin/*', '*'), ('p', '1', '/api/v1/poetry/*', '*'), ('p', '10013', '/custom/sys/menu/getMenu', 'GET'), ('g', 'user_2', '10010', '')",
		"INSERT INTO sys_data_authority_id (sys_authority_authority_id, data_authority_id_authority_id) VALUES (1, 42), (1, 10010), (10013, 42)",
		"INSERT INTO sys_operation_logs (id, path) VALUES (1, '/custom/sys/menu/getMenu'), (2, '/custom/plugin/release/claim')",
	} {
		require.NoError(t, db.Exec(statement).Error, statement)
	}

	for run := 0; run < 3; run++ {
		require.NoError(t, cleanupLegacyModules(db, "/custom"))
		for _, table := range legacyModuleTables {
			require.False(t, db.Migrator().HasTable(table), table)
		}
		for table, expected := range map[string]int64{
			"sys_users": 2, "sys_authorities": 3, "sys_menus": 2, "sys_authority_menus": 2,
			"sys_apis": 1, "sys_authority_apis": 1, "sys_api_tokens": 1, "sys_api_token_apis": 1,
			"sys_casbin_rules": 1, "sys_user_authorities": 3, "sys_data_authority_id": 1,
			"sys_operation_logs": 1, "sys_schema_migrations": 1,
		} {
			var count int64
			require.NoError(t, db.Table(table).Count(&count).Error)
			require.Equal(t, expected, count, table)
		}
		var user model.SysUser
		require.NoError(t, db.First(&user, 2).Error)
		require.Equal(t, "keep-user-hash", user.Password)
		require.Equal(t, uint(888), user.AuthorityID)
		var admin model.SysUser
		require.NoError(t, db.First(&admin, 1).Error)
		require.Equal(t, "keep-admin-hash", admin.Password)
		require.Equal(t, uint(1), admin.AuthorityID)
		var team model.SysAuthority
		require.NoError(t, db.Where("authority_id = 42").First(&team).Error)
		require.Equal(t, "dashboard/workplace", team.DefaultRouter)
		require.Zero(t, team.ParentId)
		// Simulate interruption after DDL but before recording completion.
		if run == 0 {
			require.NoError(t, db.Exec("DELETE FROM sys_schema_migrations").Error)
		}
	}
}

func TestCleanupLegacyModulesOnFreshDatabase(t *testing.T) {
	db := cleanupTestDB(t)
	require.NoError(t, cleanupLegacyModules(db, "/api/v1"))
	require.NoError(t, cleanupLegacyModules(db, "/api/v1"))
	var count int64
	require.NoError(t, db.Model(&model.SysAuthority{}).Count(&count).Error)
	require.Zero(t, count, "fresh install should use normal system seeding")
}
