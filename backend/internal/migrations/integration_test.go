package migrations_test

import (
	"context"
	"github.com/CIPFZ/video-cms/internal/core/claims"
	"github.com/CIPFZ/video-cms/internal/core/config"
	"github.com/CIPFZ/video-cms/internal/core/jwt"
	"github.com/CIPFZ/video-cms/internal/core/session"
	"github.com/CIPFZ/video-cms/internal/migrations"
	"github.com/CIPFZ/video-cms/internal/modules/system/dto"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/CIPFZ/video-cms/internal/modules/system/repository"
	"github.com/CIPFZ/video-cms/internal/modules/system/service"
	"github.com/CIPFZ/video-cms/internal/svc"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// External DSNs must point to disposable databases. CI and db-matrix.py own them.
func TestSecurityAndMigrationMatrix(t *testing.T) {
	t.Setenv("SEED_ADMIN_ENABLED", "true")
	t.Setenv("SEED_ADMIN_PASSWORD", "initial-password")
	dialects := map[string]gorm.Dialector{"sqlite": sqlite.Open(filepath.Join(t.TempDir(), "cms.db") + "?_pragma=busy_timeout(10000)")}
	if dsn := os.Getenv("TEST_MYSQL_DSN"); dsn != "" {
		dialects["mysql"] = mysql.Open(dsn)
	}
	if dsn := os.Getenv("TEST_POSTGRES_DSN"); dsn != "" {
		dialects["postgres"] = postgres.Open(dsn)
	}
	for name, dialect := range dialects {
		t.Run(name, func(t *testing.T) {
			db, err := gorm.Open(dialect, &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true, Logger: logger.Default.LogMode(logger.Silent)})
			require.NoError(t, err)
			pool, err := db.DB()
			require.NoError(t, err)
			defer pool.Close()
			pool.SetMaxOpenConns(4)
			cfg := &config.Config{System: config.System{Environment: "dev", RouterPrefix: "/api/v1"}, JWT: config.JWT{SigningKey: "a-long-signing-key-for-security-tests", Issuer: "tests", ExpiresTime: "1h", BufferTime: "5m"}}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			require.Error(t, migrations.Check(db))
			var wg sync.WaitGroup
			errs := make(chan error, 2)
			for n := 0; n < 2; n++ {
				wg.Add(1)
				go func() { defer wg.Done(); errs <- migrations.Run(ctx, db, cfg, zap.NewNop()) }()
			}
			wg.Wait()
			close(errs)
			for e := range errs {
				require.NoError(t, e)
			}
			require.NoError(t, migrations.Check(db))
			var users int64
			require.NoError(t, db.Model(&model.SysUser{}).Count(&users).Error)
			require.EqualValues(t, 1, users)
			verifyNoticesAndTokenSchema(t, db, ctx)
			managerA, err := claims.NewPolicyManager(db)
			require.NoError(t, err)
			managerB, err := claims.NewPolicyManager(db)
			require.NoError(t, err)
			policies := repository.NewCasbinRepository(db)
			require.NoError(t, policies.ReplacePolicy(ctx, "888", [][]string{{"888", "/api/v1/sys/user/getSelfInfo", "GET"}}))
			for _, manager := range []*claims.PolicyManager{managerA, managerB} {
				ok, err := manager.Enforce(ctx, "888", "/api/v1/sys/user/getSelfInfo", "GET")
				require.NoError(t, err)
				require.True(t, ok)
			}
			// Bad API references roll back both revision and grants.
			var before claims.PolicyRevision
			require.NoError(t, db.First(&before, 1).Error)
			require.Error(t, policies.ReplacePolicy(ctx, "888", [][]string{{"888", "/does-not-exist", "GET"}}))
			var after claims.PolicyRevision
			require.NoError(t, db.First(&after, 1).Error)
			require.Equal(t, before.Version, after.Version)
			require.NoError(t, policies.ReplacePolicy(ctx, "888", nil))
			for _, manager := range []*claims.PolicyManager{managerA, managerB} {
				ok, err := manager.Enforce(ctx, "888", "/api/v1/sys/user/getSelfInfo", "GET")
				require.NoError(t, err)
				require.False(t, ok)
			}
			require.NoError(t, db.Model(&model.SysMenu{}).Where("path = ?", "/about").Updates(map[string]any{"component": "custom", "hide_in_menu": true, "sort": 999}).Error)
			require.NoError(t, migrations.Run(ctx, db, cfg, zap.NewNop()))
			var menu model.SysMenu
			require.NoError(t, db.Where("path = ?", "/about").First(&menu).Error)
			require.Equal(t, "custom", menu.Component)
			require.True(t, menu.HideInMenu)
			require.Equal(t, 999, menu.Sort)
			remaining, err := policies.GetPolicy(ctx, "888")
			require.NoError(t, err)
			require.Empty(t, remaining)
			menus := repository.NewMenuRepository(db)
			rootMenu := model.SysMenu{Path: "/test-root"}
			require.NoError(t, menus.Create(ctx, &rootMenu))
			child := model.SysMenu{Path: "/test-child", ParentId: rootMenu.ID}
			require.NoError(t, menus.Create(ctx, &child))
			require.Error(t, menus.Update(ctx, &rootMenu, map[string]any{"parent_id": child.ID}))
			require.Error(t, menus.Create(ctx, &model.SysMenu{Path: "/test-root"}))
			require.Error(t, menus.DeleteWithAssociations(ctx, rootMenu.ID))
			require.Error(t, repository.NewAuthorityRepository(db).SetMenuAuthority(ctx, 888, []uint{999999}))

			sessions := session.NewStore(db)
			sc := svc.NewServiceContext()
			sc.DB = db
			sc.Config = cfg
			sc.Logger = zap.NewNop()
			sc.Sessions = sessions
			sc.JWT = jwt.NewJWT(cfg.JWT, sc.Logger, nil)
			repo := repository.NewUserRepository(db)
			usersvc := service.NewUserService(sc, repo)
			_, err = usersvc.Register(ctx, dto.RegisterReq{Username: "public", Password: "password123"})
			require.Error(t, err)
			cfg.System.AllowRegistration = true
			public, err := usersvc.Register(ctx, dto.RegisterReq{Username: "public", Password: "password123"})
			require.NoError(t, err)
			require.EqualValues(t, 888, public.AuthorityID)
			require.Error(t, usersvc.AddUser(ctx, dto.AddUserReq{Username: "bad-role", Password: "password123", AuthorityIds: []uint{999999}}))
			login := func() session.Identity {
				res, err := usersvc.Login(ctx, dto.LoginReq{Username: "public", Password: "password123"})
				require.NoError(t, err)
				c, err := sc.JWT.ParseToken(res.Token)
				require.NoError(t, err)
				identity := session.Identity{ID: c.ID, UserID: c.UserID, Version: c.TokenVersion, AuthorityID: c.AuthorityId}
				require.NoError(t, sessions.Validate(ctx, identity))
				return identity
			}
			first, other := login(), login()
			expiry := time.Now().Add(2 * time.Hour).Truncate(time.Second)
			require.NoError(t, sessions.Renew(ctx, first, expiry))
			require.NoError(t, sessions.Renew(ctx, first, expiry))
			require.NoError(t, sessions.Renew(ctx, first, expiry.Add(-time.Minute)))
			require.NoError(t, sessions.Revoke(ctx, first.ID))
			require.Error(t, sessions.Validate(ctx, first))
			require.NoError(t, sessions.Validate(ctx, other))
			require.NoError(t, usersvc.ResetPassword(ctx, dto.ResetPasswordReq{ID: public.ID, Password: "password123"}))
			require.Error(t, sessions.Validate(ctx, other))
			token := login()
			require.NoError(t, usersvc.UpdateUser(ctx, dto.UpdateUserReq{ID: public.ID, AuthorityIds: []uint{888}, Status: 0}))
			require.Error(t, sessions.Validate(ctx, token))
			require.NoError(t, usersvc.UpdateUser(ctx, dto.UpdateUserReq{ID: public.ID, AuthorityIds: []uint{888}, Status: 1}))
			require.Error(t, sessions.Validate(ctx, token))
			token = login()
			require.NoError(t, usersvc.UpdateUser(ctx, dto.UpdateUserReq{ID: public.ID, AuthorityIds: []uint{888, 8881}, Status: 1}))
			require.Error(t, sessions.Validate(ctx, token))
			token = login()
			require.NoError(t, usersvc.DeleteUser(ctx, public.ID))
			require.Error(t, sessions.Validate(ctx, token))
			var admin model.SysUser
			require.NoError(t, db.Where("username = ?", "admin").First(&admin).Error)
			require.Error(t, usersvc.DeleteUser(ctx, admin.ID))
			require.Error(t, usersvc.UpdateUser(ctx, dto.UpdateUserReq{ID: admin.ID, AuthorityIds: []uint{888}, Status: 1}))
		})
	}
}

func verifyNoticesAndTokenSchema(t *testing.T, db *gorm.DB, ctx context.Context) {
	t.Helper()
	var admin model.SysUser
	require.NoError(t, db.Where("username = ?", "admin").First(&admin).Error)
	notices := service.NewNoticeService(repository.NewNoticeRepository(db))
	req := dto.CreateNoticeReq{Title: "Matrix notice", Content: "database portability", TargetType: model.NoticeTargetUsers, TargetIDs: []uint{admin.ID, admin.ID}, IsPopup: true}
	require.NoError(t, notices.CreateNotice(ctx, req, admin.ID))
	list, _, err := notices.GetNoticeList(ctx, dto.SearchNoticeReq{})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, []uint{admin.ID}, list[0].TargetIDs)
	require.EqualValues(t, 1, list[0].ReceiverCount)
	id := list[0].ID
	require.Error(t, notices.MarkRead(ctx, id, 999999))
	require.NoError(t, notices.MarkRead(ctx, id, admin.ID))
	require.NoError(t, notices.MarkRead(ctx, id, admin.ID))
	popup, count, err := notices.GetMyNotices(ctx, admin.ID, 1, 100, true)
	require.NoError(t, err)
	require.Empty(t, popup)
	require.Zero(t, count)
	require.NoError(t, db.Model(&model.SysNotice{}).Where("id = ?", id).Update("end_at", time.Now().Add(-time.Hour)).Error)
	require.Error(t, notices.MarkRead(ctx, id, admin.ID))
	var external model.SysApi
	require.NoError(t, db.Where("path = ?", "/api/v1/open/token-info").First(&external).Error)
	tokens := service.NewApiTokenService(repository.NewApiTokenRepository(db))
	expiry := time.Now().Add(time.Hour).Format(time.RFC3339)
	created, err := tokens.CreateApiToken(ctx, admin.ID, dto.CreateApiTokenReq{Name: "matrix", ExpiresAt: &expiry, MaxConcurrency: 1, ApiIds: []uint{external.ID}})
	require.NoError(t, err)
	require.NoError(t, tokens.DisableApiToken(ctx, created.ID))
	require.NoError(t, tokens.DisableApiToken(ctx, created.ID))
	_, err = tokens.ResetApiToken(ctx, created.ID)
	require.NoError(t, err)
	require.NoError(t, tokens.DeleteApiToken(ctx, dto.DeleteApiTokenReq{ID: created.ID}))
	require.Error(t, tokens.EnableApiToken(ctx, created.ID))
}
