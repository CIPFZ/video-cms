package server

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/CIPFZ/video-cms/internal/core/config"
	"github.com/CIPFZ/video-cms/internal/core/db"
	"github.com/CIPFZ/video-cms/internal/modules/media"
	"github.com/CIPFZ/video-cms/internal/svc"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPublicMediaNeedsNoLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database, err := db.InitDatabase(config.Database{Driver: "sqlite3", SQLite: config.SQLite{Path: filepath.Join(t.TempDir(), "public.db"), MaxOpenConns: 1, MaxIdleConns: 1}}, zap.NewNop())
	require.NoError(t, err)
	sqlDB, _ := database.DB()
	t.Cleanup(func() { sqlDB.Close() })
	require.NoError(t, database.AutoMigrate(&media.Source{}, &media.Item{}, &media.SourceItem{}, &media.PlayLine{}, &media.PlayEpisode{}))
	collector, err := media.NewService(media.Config{DB: database})
	require.NoError(t, err)
	sc := svc.NewServiceContext()
	sc.DB = database
	sc.Logger = zap.NewNop()
	sc.MediaCollector = collector
	sc.Config = &config.Config{System: config.System{RouterPrefix: "/api/v1"}, Observable: config.Observability{Exporter: "none"}}
	router := InitRouters(sc)

	get := func(path string) (int, int) {
		t.Helper()
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		var body struct{ Code int }
		if w.Code == 200 {
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		}
		return w.Code, body.Code
	}
	for _, path := range []string{"/api/v1/public/media/home", "/api/v1/public/media/items?type=movie&year=2023,none", "/api/v1/public/media/facets"} {
		status, code := get(path)
		require.Equal(t, 200, status, path)
		require.Zero(t, code, path)
	}
	// Twenty-one quick reads would trip the login quota if these routes shared it.
	for range 21 {
		status, _ := get("/api/v1/public/media/items")
		require.Equal(t, 200, status)
	}
	status, _ := get("/api/v1/public/media/episodes/1/stream")
	require.Equal(t, 404, status)
	_, code := get("/api/v1/media/items")
	require.Equal(t, 1003, code, "the back-office list still needs login")
}
