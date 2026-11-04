package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/CIPFZ/video-cms/internal/core/config"
	"github.com/CIPFZ/video-cms/internal/core/db"
	tokenCore "github.com/CIPFZ/video-cms/internal/core/token"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/CIPFZ/video-cms/internal/svc"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestApiTokenAuthRejectsMissingToken(t *testing.T) {
	engine, _ := newAPITokenMiddlewareTestEngine(t, func(group *gin.RouterGroup) {
		group.GET("test/resources", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"code": 0})
		})
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test/resources", nil)

	engine.ServeHTTP(rec, req)

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response error = %v", err)
	}
	if int(body["code"].(float64)) != 1003 {
		t.Fatalf("response code = %v, want 1003", body["code"])
	}
}

func TestApiTokenAuthAllowsAuthorizedRoute(t *testing.T) {
	rawToken := "cms_allow_token"
	engine, _ := newAPITokenMiddlewareTestEngine(t, func(group *gin.RouterGroup) {
		group.GET("test/resources", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"code": 0, "apiTokenId": c.GetUint(CtxKeyAPITokenID)})
		})
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test/resources", nil)
	req.Header.Set("X-API-Token", rawToken)

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response error = %v", err)
	}
	if int(body["code"].(float64)) != 0 {
		t.Fatalf("response code = %v, want 0", body["code"])
	}
	if int(body["apiTokenId"].(float64)) == 0 {
		t.Fatalf("apiTokenId = %v, want non-zero", body["apiTokenId"])
	}
}

func TestApiTokenAuthRejectsUnauthorizedRoute(t *testing.T) {
	rawToken := "cms_forbidden_token"
	engine, _ := newAPITokenMiddlewareTestEngine(t, func(group *gin.RouterGroup) {
		group.GET("test/other-resources", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"code": 0})
		})
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test/other-resources", nil)
	req.Header.Set("X-API-Token", rawToken)

	engine.ServeHTTP(rec, req)

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response error = %v", err)
	}
	if int(body["code"].(float64)) != 1004 {
		t.Fatalf("response code = %v, want 1004", body["code"])
	}
}

func newAPITokenMiddlewareTestEngine(t *testing.T, registerRoutes func(group *gin.RouterGroup)) (*gin.Engine, *svc.ServiceContext) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	gormDB, err := db.InitDatabase(config.Database{
		Driver: "sqlite3",
		SQLite: config.SQLite{
			Path:         filepath.Join(t.TempDir(), "middleware.db"),
			MaxIdleConns: 1,
			MaxOpenConns: 1,
		},
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("InitDatabase() error = %v", err)
	}
	if err := gormDB.AutoMigrate(
		&model.SysApi{},
		&model.SysApiToken{},
		&model.SysApiTokenApi{},
	); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	api := model.SysApi{
		Path:        "/api/v1/test/resources",
		Method:      "GET",
		ApiGroup:    "test",
		Description: "List resources",
	}
	if err := gormDB.Create(&api).Error; err != nil {
		t.Fatalf("create sys api error = %v", err)
	}

	expiresAt := time.Now().Add(time.Hour)
	token := model.SysApiToken{
		TokenHash:      tokenCore.HashToken("cms_allow_token"),
		TokenPrefix:    "cms_allo",
		Name:           "allowed",
		MaxConcurrency: 1,
		Enabled:        true,
		ExpiresAt:      &expiresAt,
		Apis:           []model.SysApi{api},
	}
	if err := gormDB.Create(&token).Error; err != nil {
		t.Fatalf("create api token error = %v", err)
	}

	forbidden := model.SysApiToken{
		TokenHash:      tokenCore.HashToken("cms_forbidden_token"),
		TokenPrefix:    "cms_forb",
		Name:           "forbidden",
		MaxConcurrency: 1,
		Enabled:        true,
		ExpiresAt:      &expiresAt,
	}
	if err := gormDB.Create(&forbidden).Error; err != nil {
		t.Fatalf("create forbidden api token error = %v", err)
	}

	svcCtx := svc.NewServiceContext()
	svcCtx.DB = gormDB
	svcCtx.Logger = zap.NewNop()

	engine := gin.New()
	group := engine.Group("/api/v1")
	group.Use(ApiTokenAuth(svcCtx))
	registerRoutes(group)
	return engine, svcCtx
}

func TestApiTokenConcurrencyReleasedAfterHandler(t *testing.T) {
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	engine, _ := newAPITokenMiddlewareTestEngine(t, func(group *gin.RouterGroup) {
		group.GET("test/resources", func(c *gin.Context) { entered <- struct{}{}; <-release; c.JSON(200, gin.H{"code": 0}) })
	})
	call := func() int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/test/resources", nil)
		req.Header.Set("X-API-Token", "cms_allow_token")
		engine.ServeHTTP(w, req)
		return w.Code
	}
	done := make(chan int, 1)
	go func() { done <- call() }()
	<-entered
	if status := call(); status != 429 {
		t.Fatalf("concurrent status = %d", status)
	}
	close(release)
	if status := <-done; status != 200 {
		t.Fatal(status)
	}
	if status := call(); status != 200 {
		t.Fatalf("quota not released: %d", status)
	}
}
