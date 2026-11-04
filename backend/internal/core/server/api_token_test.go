package server

import (
	"encoding/json"
	"github.com/CIPFZ/video-cms/internal/core/config"
	"github.com/CIPFZ/video-cms/internal/core/db"
	tokenCore "github.com/CIPFZ/video-cms/internal/core/token"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/CIPFZ/video-cms/internal/svc"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestActualRouterTokenBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, err := db.InitDatabase(config.Database{Driver: "sqlite3", SQLite: config.SQLite{Path: filepath.Join(t.TempDir(), "router.db"), MaxOpenConns: 1, MaxIdleConns: 1}}, zap.NewNop())
	require.NoError(t, err)
	sqlDB, _ := database.DB()
	t.Cleanup(func() { sqlDB.Close() })
	require.NoError(t, database.AutoMigrate(&model.SysApi{}, &model.SysApiToken{}, &model.SysApiTokenApi{}))
	sc := svc.NewServiceContext()
	sc.DB = database
	sc.Logger = zap.NewNop()
	sc.Config = &config.Config{System: config.System{RouterPrefix: "/api/v1"}, Observable: config.Observability{Exporter: "none"}}
	router := InitRouters(sc)
	path := "/api/v1/open/token-info"
	api := model.SysApi{Path: path, Method: "GET"}
	require.NoError(t, database.Create(&api).Error)
	expires := time.Now().Add(time.Hour)
	token := model.SysApiToken{Name: "test", TokenHash: tokenCore.HashToken("secret"), Enabled: true, MaxConcurrency: 1, ExpiresAt: &expires, Apis: []model.SysApi{api}}
	require.NoError(t, database.Create(&token).Error)
	call := func(method, path, header, value string) (int, int) {
		t.Helper()
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set(header, value)
		router.ServeHTTP(w, req)
		var body struct{ Code int }
		if w.Code == 200 {
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		}
		return w.Code, body.Code
	}
	assertCode := func(path, header, value string, code int) {
		t.Helper()
		status, got := call("GET", path, header, value)
		require.Equal(t, 200, status)
		require.Equal(t, code, got)
	}
	assertCode(path, "X-API-Token", "secret", 0)
	assertCode(path, "X-API-Token", "wrong", 1003)
	assertCode(path, "x-token", "secret", 1003)
	assertCode("/api/v1/sys/user/getSelfInfo", "X-API-Token", "secret", 1003)
	status, _ := call("POST", path, "X-API-Token", "secret")
	require.Equal(t, 404, status)
	require.NoError(t, database.Model(&token).Update("enabled", false).Error)
	assertCode(path, "X-API-Token", "secret", 1003)
	require.NoError(t, database.Model(&token).Updates(map[string]any{"enabled": true, "expires_at": time.Now().Add(-time.Second)}).Error)
	assertCode(path, "X-API-Token", "secret", 1003)
	require.NoError(t, database.Model(&token).Updates(map[string]any{"expires_at": time.Now().Add(time.Hour), "token_hash": tokenCore.HashToken("new-secret")}).Error)
	assertCode(path, "X-API-Token", "secret", 1003)
	assertCode(path, "X-API-Token", "new-secret", 0)
	require.NoError(t, database.Model(&api).Update("method", "POST").Error)
	assertCode(path, "X-API-Token", "new-secret", 1004)
	require.NoError(t, database.Model(&api).Update("method", "GET").Error)
	require.NoError(t, database.Where("api_token_id = ?", token.ID).Delete(&model.SysApiTokenApi{}).Error)
	assertCode(path, "X-API-Token", "new-secret", 1004)
	require.NoError(t, database.Delete(&token).Error)
	assertCode(path, "X-API-Token", "new-secret", 1003)
}
