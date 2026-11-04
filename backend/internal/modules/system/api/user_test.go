package api

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CIPFZ/video-cms/internal/core/config"
	"github.com/CIPFZ/video-cms/internal/svc"
	"github.com/gin-gonic/gin"
)

func TestSetTokenHelperSetsSameSiteCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	api := &UserApi{
		svcCtx: &svc.ServiceContext{
			Config: &config.Config{
				System: config.System{Environment: "dev"},
			},
		},
	}

	api.setTokenHelper(ctx, "token-value", 60)

	cookies := recorder.Header().Values("Set-Cookie")
	if len(cookies) == 0 {
		t.Fatal("expected Set-Cookie header to be written")
	}
	if !strings.Contains(cookies[0], "SameSite=Lax") {
		t.Fatalf("Set-Cookie = %q, want SameSite=Lax", cookies[0])
	}
}
