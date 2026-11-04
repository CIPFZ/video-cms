package service

import (
	"context"
	"github.com/CIPFZ/video-cms/internal/core/claims"
	"path/filepath"
	"testing"
	"time"

	"github.com/CIPFZ/video-cms/internal/core/config"
	"github.com/CIPFZ/video-cms/internal/core/db"
	tokenCore "github.com/CIPFZ/video-cms/internal/core/token"
	"github.com/CIPFZ/video-cms/internal/modules/system/dto"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/CIPFZ/video-cms/internal/modules/system/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestApiTokenServiceCreatePersistsHashAndApis(t *testing.T) {
	gormDB := newApiTokenTestDB(t)
	if err := gormDB.Create(&model.SysApi{
		Path:        "/api/v1/open/token-info",
		Method:      "GET",
		ApiGroup:    "test",
		Description: "List resources",
	}).Error; err != nil {
		t.Fatalf("seed sys api error = %v", err)
	}

	var api model.SysApi
	if err := gormDB.First(&api).Error; err != nil {
		t.Fatalf("load seeded api error = %v", err)
	}

	service := NewApiTokenService(
		repository.NewApiTokenRepository(gormDB),
	)

	expires := time.Now().Add(time.Hour).Format(time.RFC3339)
	resp, err := service.CreateApiToken(context.Background(), 99, dto.CreateApiTokenReq{
		Name:           "test-reader",
		ExpiresAt:      &expires,
		Description:    "read test api",
		MaxConcurrency: 2,
		ApiIds:         []uint{api.ID},
	})
	if err != nil {
		t.Fatalf("CreateApiToken() error = %v", err)
	}
	if resp.Token == "" {
		t.Fatal("CreateApiToken() returned empty raw token")
	}
	if len(resp.Apis) != 1 || resp.Apis[0].ID != api.ID {
		t.Fatalf("CreateApiToken() apis = %#v, want api id %d", resp.Apis, api.ID)
	}

	var stored model.SysApiToken
	if err := gormDB.Preload("Apis").First(&stored).Error; err != nil {
		t.Fatalf("load stored api token error = %v", err)
	}
	if stored.TokenHash == resp.Token {
		t.Fatal("stored token hash should not equal raw token")
	}
	if !tokenCore.VerifyToken(resp.Token, stored.TokenHash) {
		t.Fatal("stored token hash does not match returned raw token")
	}
	if stored.CreatedBy != 99 {
		t.Fatalf("stored created_by = %d, want 99", stored.CreatedBy)
	}
	if len(stored.Apis) != 1 || stored.Apis[0].ID != api.ID {
		t.Fatalf("stored apis = %#v, want api id %d", stored.Apis, api.ID)
	}
}

func TestApiTokenServiceResetReplacesStoredHash(t *testing.T) {
	gormDB := newApiTokenTestDB(t)
	if err := gormDB.Create(&model.SysApi{
		Path:        "/api/v1/open/token-info",
		Method:      "GET",
		ApiGroup:    "test",
		Description: "List records",
	}).Error; err != nil {
		t.Fatalf("seed sys api error = %v", err)
	}

	var api model.SysApi
	if err := gormDB.First(&api).Error; err != nil {
		t.Fatalf("load seeded api error = %v", err)
	}

	service := NewApiTokenService(
		repository.NewApiTokenRepository(gormDB),
	)

	expires := time.Now().Add(time.Hour).Format(time.RFC3339)
	created, err := service.CreateApiToken(context.Background(), 7, dto.CreateApiTokenReq{
		Name:           "reset-me",
		ExpiresAt:      &expires,
		MaxConcurrency: 1,
		ApiIds:         []uint{api.ID},
	})
	if err != nil {
		t.Fatalf("CreateApiToken() error = %v", err)
	}

	var before model.SysApiToken
	if err := gormDB.First(&before).Error; err != nil {
		t.Fatalf("load before reset error = %v", err)
	}

	resetResp, err := service.ResetApiToken(context.Background(), before.ID)
	if err != nil {
		t.Fatalf("ResetApiToken() error = %v", err)
	}
	if resetResp.Token == "" {
		t.Fatal("ResetApiToken() returned empty raw token")
	}
	if resetResp.Token == created.Token {
		t.Fatal("ResetApiToken() returned old token, want new token")
	}

	var after model.SysApiToken
	if err := gormDB.First(&after, before.ID).Error; err != nil {
		t.Fatalf("load after reset error = %v", err)
	}
	if before.TokenHash == after.TokenHash {
		t.Fatal("ResetApiToken() did not replace token hash")
	}
	if tokenCore.VerifyToken(created.Token, after.TokenHash) {
		t.Fatal("old token should not match new stored hash")
	}
	if !tokenCore.VerifyToken(resetResp.Token, after.TokenHash) {
		t.Fatal("new token does not match stored hash")
	}
}

func newApiTokenTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gormDB, err := db.InitDatabase(config.Database{
		Driver: "sqlite3",
		SQLite: config.SQLite{
			Path:         filepath.Join(t.TempDir(), "apitoken.db"),
			MaxIdleConns: 1,
			MaxOpenConns: 1,
		},
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("InitDatabase() error = %v", err)
	}

	if err := gormDB.AutoMigrate(
		&model.SysApi{}, &claims.PolicyRevision{},
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

	if err := gormDB.Create(&claims.PolicyRevision{ID: 1, Version: 1}).Error; err != nil {
		t.Fatal(err)
	}
	return gormDB
}

func TestApiTokenServiceCreateRejectsExpiredTimeInPast(t *testing.T) {
	gormDB := newApiTokenTestDB(t)
	service := NewApiTokenService(
		repository.NewApiTokenRepository(gormDB),
	)

	past := time.Now().Add(-time.Hour).Format(time.RFC3339)
	_, err := service.CreateApiToken(context.Background(), 1, dto.CreateApiTokenReq{
		Name:        "expired",
		ExpiresAt:   &past,
		ApiIds:      []uint{},
		NeverExpire: false,
	})
	if err == nil {
		t.Fatal("CreateApiToken() error = nil, want validation error")
	}
}

func TestApiTokenRejectsCMSPermission(t *testing.T) {
	database := newApiTokenTestDB(t)
	api := model.SysApi{Path: "/api/v1/sys/user/getUserList", Method: "POST"}
	database.Create(&api)
	expires := time.Now().Add(time.Hour).Format(time.RFC3339)
	service := NewApiTokenService(repository.NewApiTokenRepository(database))
	_, err := service.CreateApiToken(context.Background(), 1, dto.CreateApiTokenReq{Name: "bad grant", ExpiresAt: &expires, MaxConcurrency: 1, ApiIds: []uint{api.ID}})
	if err == nil {
		t.Fatal("JWT-only CMS API accepted as token permission")
	}
}

func TestApiTokenToggleMissingFails(t *testing.T) {
	service := NewApiTokenService(repository.NewApiTokenRepository(newApiTokenTestDB(t)))
	if err := service.DisableApiToken(context.Background(), 999); err == nil {
		t.Fatal("nonexistent token reported success")
	}
}

// Deterministically interleave permission removal between reset's read and write.
type revokeOnReadRepo struct {
	repository.IApiTokenRepository
	database *gorm.DB
}

func (r revokeOnReadRepo) FindByID(ctx context.Context, id uint) (*model.SysApiToken, error) {
	token, err := r.IApiTokenRepository.FindByID(ctx, id)
	if err == nil {
		err = r.database.Where("api_token_id = ?", id).Delete(&model.SysApiTokenApi{}).Error
	}
	return token, err
}
func TestResetDoesNotRestoreConcurrentlyRevokedPermissions(t *testing.T) {
	database := newApiTokenTestDB(t)
	api := model.SysApi{Path: "/api/v1/open/token-info", Method: "GET"}
	database.Create(&api)
	expires := time.Now().Add(time.Hour).Format(time.RFC3339)
	repo := repository.NewApiTokenRepository(database)
	service := NewApiTokenService(repo)
	created, err := service.CreateApiToken(context.Background(), 1, dto.CreateApiTokenReq{Name: "rotate", ExpiresAt: &expires, MaxConcurrency: 1, ApiIds: []uint{api.ID}})
	if err != nil {
		t.Fatal(err)
	}
	service = NewApiTokenService(revokeOnReadRepo{repo, database})
	if _, err := service.ResetApiToken(context.Background(), created.ID); err != nil {
		t.Fatal(err)
	}
	stored, err := repo.FindByID(context.Background(), created.ID)
	if err != nil || len(stored.Apis) != 0 {
		t.Fatalf("revoked permissions restored: %v %v", stored, err)
	}
}
