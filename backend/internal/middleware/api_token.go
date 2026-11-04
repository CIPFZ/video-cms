package middleware

import (
	"errors"
	"strings"
	"time"

	tokenCore "github.com/CIPFZ/video-cms/internal/core/token"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/CIPFZ/video-cms/internal/svc"
	"github.com/CIPFZ/video-cms/pkg/errcode"
	"github.com/CIPFZ/video-cms/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	CtxKeyAPITokenID = "apiTokenId"
)

func ApiTokenAuth(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	if svcCtx.APITokenLimiter == nil {
		svcCtx.APITokenLimiter = tokenCore.NewInMemoryLimiter()
	}

	return func(c *gin.Context) {
		rawToken := strings.TrimSpace(c.GetHeader("X-API-Token"))
		if rawToken == "" {
			response.FailWithCode(errcode.Unauthorized, c)
			c.Abort()
			return
		}

		var token model.SysApiToken
		err := svcCtx.DB.WithContext(c.Request.Context()).
			Preload("Apis").
			Where("token_hash = ?", tokenCore.HashToken(rawToken)).
			First(&token).
			Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				response.FailWithCode(errcode.Unauthorized, c)
				c.Abort()
				return
			}
			response.FailWithCode(errcode.ServerError, c)
			c.Abort()
			return
		}

		if !token.Enabled {
			response.FailWithCode(errcode.Unauthorized.WithDetails("token 已禁用"), c)
			c.Abort()
			return
		}
		if token.ExpiresAt != nil && !token.ExpiresAt.After(time.Now()) {
			response.FailWithCode(errcode.Unauthorized.WithDetails("token 已过期"), c)
			c.Abort()
			return
		}

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		if !apiTokenAllowsRequest(&token, c.Request.Method, path) {
			response.FailWithCode(errcode.AssessDenied.WithDetails("token 无权访问该 API"), c)
			c.Abort()
			return
		}

		if svcCtx.Redis != nil {
			ctx, release, err := tokenCore.AcquireRedisLease(c.Request.Context(), svcCtx.Redis, token.ID, token.MaxConcurrency)
			if err != nil {
				if errors.Is(err, tokenCore.ErrQuota) {
					c.AbortWithStatusJSON(429, response.Response{Code: 429, Msg: response.LocalizeMessage(c, "token.quota")})
				} else {
					c.AbortWithStatusJSON(503, response.Response{Code: 503, Msg: response.LocalizeMessage(c, "token.limiterUnavailable")})
				}
				return
			}
			defer release()
			c.Request = c.Request.WithContext(ctx)
		} else {
			if !svcCtx.APITokenLimiter.Acquire(token.ID, token.MaxConcurrency) {
				c.AbortWithStatusJSON(429, response.Response{Code: 429, Msg: response.LocalizeMessage(c, "token.quota")})
				return
			}
			defer svcCtx.APITokenLimiter.Release(token.ID)
		}

		c.Set(CtxKeyAPITokenID, token.ID)
		c.Next()

		_ = svcCtx.DB.WithContext(c.Request.Context()).
			Model(&model.SysApiToken{}).
			Where("id = ?", token.ID).
			Update("last_used_at", time.Now()).
			Error
	}
}

func apiTokenAllowsRequest(token *model.SysApiToken, method, path string) bool {
	for _, api := range token.Apis {
		if strings.EqualFold(api.Method, method) && api.Path == path {
			return true
		}
	}
	return false
}
