package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CIPFZ/video-cms/internal/core/config"
	"github.com/CIPFZ/video-cms/internal/modules/system/service"
	"github.com/CIPFZ/video-cms/internal/svc"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

type avatarUserService struct {
	service.IUserService
	avatar string
}

func (s *avatarUserService) UpdateAvatar(_ context.Context, _ uuid.UUID, avatar string) error {
	s.avatar = avatar
	return nil
}

func TestUpdateAvatarAcceptsOnlyPresets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		body, saved string
	}{
		{`{"avatar":"/avatars/cat.svg"}`, "/avatars/cat.svg"},
		{`{"avatar":"/uploads/file/evil.svg"}`, ""},
		{`{"avatar":"/avatars/../logo.svg"}`, ""},
		{`{}`, ""},
	} {
		users := &avatarUserService{}
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/sys/user/avatar", strings.NewReader(tc.body))
		ctx.Request.Header.Set("Content-Type", "application/json")

		(&UserApi{userService: users}).UpdateAvatar(ctx)

		var resp struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
			t.Fatalf("%s: decode response: %v", tc.body, err)
		}
		if ok := resp.Code == 0; ok != (tc.saved != "") || users.avatar != tc.saved {
			t.Fatalf("%s: code=%d saved=%q, want saved=%q", tc.body, resp.Code, users.avatar, tc.saved)
		}
	}
}
