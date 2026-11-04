package repository

import (
	"context"
	"github.com/CIPFZ/video-cms/internal/core/claims"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"testing"
)

func TestUpdateAPIPreservesOldIdentityForPolicySync(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	pool, _ := db.DB()
	pool.SetMaxOpenConns(1)
	defer pool.Close()
	require.NoError(t, db.AutoMigrate(&model.SysApi{}, &model.SysCasbinRule{}, &claims.PolicyRevision{}))
	require.NoError(t, db.Create(&claims.PolicyRevision{ID: 1, Version: 1}).Error)
	old := model.SysApi{Path: "/old", Method: "GET"}
	require.NoError(t, db.Create(&old).Error)
	rule := model.SysCasbinRule{Ptype: "p", V0: "2", V1: "/old", V2: "GET"}
	require.NoError(t, db.Create(&rule).Error)
	updated := model.SysApi{Path: "/new", Method: "POST"}
	updated.ID = old.ID
	require.NoError(t, NewApiRepository(db).UpdateWithSyncCasbin(context.Background(), &old, updated))
	require.NoError(t, db.First(&rule).Error)
	require.Equal(t, "/new", rule.V1)
	require.Equal(t, "POST", rule.V2)
}
