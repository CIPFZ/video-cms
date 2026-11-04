package middleware

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CIPFZ/video-cms/internal/core/audit"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/CIPFZ/video-cms/internal/svc"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestOperationRecordPreservesBodiesAndRedactsAudit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	defer sqlDB.Close()
	require.NoError(t, db.AutoMigrate(&model.SysOperationLog{}))
	recorder := audit.NewAuditRecorder(db, zap.NewNop())
	r := gin.New()
	r.Use(OperationRecord(&svc.ServiceContext{AuditRecorder: recorder}))
	r.POST("/edit", func(c *gin.Context) {
		b, err := io.ReadAll(c.Request.Body)
		require.NoError(t, err)
		c.Data(200, "application/json", b)
	})
	for _, body := range []string{`{"password":"private-value","name":"kept"}`, `{"payload":"` + strings.Repeat("x", audit.MaxBodySize*4) + `"}`} {
		req := httptest.NewRequest(http.MethodPost, "/edit", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, body, w.Body.String())
	}
	require.NoError(t, recorder.Close(context.Background()))
	var rows []model.SysOperationLog
	require.NoError(t, db.Order("id").Find(&rows).Error)
	require.Len(t, rows, 2)
	require.NotContains(t, rows[0].Body, "private-value")
	require.NotContains(t, rows[0].Resp, "private-value")
	require.Contains(t, rows[0].Body, "kept")
	require.Equal(t, "[body omitted: too large]", rows[1].Body)
	var buffer boundedCapture
	_, err = buffer.Write([]byte(strings.Repeat("x", audit.MaxBodySize*4)))
	require.NoError(t, err)
	require.Equal(t, audit.MaxBodySize+1, buffer.Len())
}
