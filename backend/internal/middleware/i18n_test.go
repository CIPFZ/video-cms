package middleware

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/CIPFZ/video-cms/internal/core/config"
	corei18n "github.com/CIPFZ/video-cms/internal/core/i18n"
	"github.com/CIPFZ/video-cms/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRequestLanguage(t *testing.T) {
	for header, want := range map[string]string{"": "zh-CN", "en": "en-US", "en-GB": "en-US", "zh-TW": "zh-CN", "zh-CN": "zh-CN", "fr": "zh-CN", "zh;q=0.3,en;q=0.9": "en-US", "en;q=0": "zh-CN"} {
		require.Equal(t, want, RequestLanguage(header), header)
	}
}

func TestResponsesAreLocalizedPerRequest(t *testing.T) {
	service, err := corei18n.NewI18n(config.I18n{Path: "../../configs/locales"}, zap.NewNop())
	require.NoError(t, err)
	router := gin.New()
	router.Use(I18nMiddleware(service))
	router.GET("/success", func(c *gin.Context) { response.OkWithMessage("成功", c) })
	router.GET("/business", func(c *gin.Context) { response.FailWithMessage("创建失败: API 不存在", c) })
	router.GET("/unknown", func(c *gin.Context) { response.FailWithMessage("driver: password=private", c) })
	router.GET("/auth", func(c *gin.Context) { response.FailWithMessage("会话已失效，请重新登录", c) })
	router.POST("/validation", func(c *gin.Context) {
		var input struct {
			Username string `json:"username" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			response.FailWithValidation(err, c)
		}
	})
	cases := []struct{ path, lang, text string }{
		{"/success", "en", "Success"}, {"/success", "zh", "成功"},
		{"/business", "en", "Create failed: API does not exist"}, {"/business", "zh", "创建失败: API 不存在"},
		{"/unknown", "en", "Internal server error"}, {"/unknown", "zh", "服务内部错误"},
		{"/auth", "en", "Your session has expired. Please sign in again."},
		{"/validation", "en", "Username is required"}, {"/validation", "zh", "用户名不能为空"},
	}
	var wg sync.WaitGroup
	for _, tc := range cases {
		wg.Add(1)
		go func() {
			defer wg.Done()
			method := "GET"
			if tc.path == "/validation" {
				method = "POST"
			}
			req := httptest.NewRequest(method, tc.path, strings.NewReader("{}"))
			req.Header.Set("Accept-Language", tc.lang)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			var body response.Response
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Error(err)
				return
			}
			if body.Msg != tc.text {
				t.Errorf("%s %s: got %q, want %q", tc.path, tc.lang, body.Msg, tc.text)
			}
			if w.Header().Get("Content-Language") != RequestLanguage(tc.lang) {
				t.Error("incorrect response language")
			}
		}()
	}
	wg.Wait()
}
