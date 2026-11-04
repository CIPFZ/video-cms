package middleware

import (
	"errors"
	"fmt"
	"github.com/CIPFZ/video-cms/pkg/response"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sony/gobreaker"
	"go.uber.org/zap"
)

// BreakerMiddleware 熔断中间件
func BreakerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	// 配置熔断规则 (企业级通常放在 Config 中，这里使用经典默认值)
	settings := gobreaker.Settings{
		Name:        "HTTP-API-Breaker",
		MaxRequests: 0, // 半开状态下允许的请求数 (0 表示默认 1)
		Interval:    0, // 计数周期 (默认 60s 清零)
		Timeout:     30 * time.Second,

		// 触发熔断的条件：
		// 当请求总数 >= 5 且 失败率 >= 60% 时，触发熔断
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 5 && failureRatio >= 0.6
		},

		// 状态变更回调 (记录日志)
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			if to == gobreaker.StateOpen {
				logger.Error("熔断器已触发 (OPEN)！系统正在进行降级保护...",
					zap.String("name", name),
					zap.String("from", from.String()))
			}
			if to == gobreaker.StateClosed && from == gobreaker.StateHalfOpen {
				logger.Info("熔断器已恢复 (CLOSED)，服务恢复正常。",
					zap.String("name", name))
			}
		},
	}

	cb := gobreaker.NewCircuitBreaker(settings)

	return func(c *gin.Context) {
		// 使用 Execute 包装请求
		_, err := cb.Execute(func() (interface{}, error) {
			c.Next()

			// 检查执行结果
			status := c.Writer.Status()
			if status >= 500 {
				// 返回错误，计入熔断器失败次数
				return nil, fmt.Errorf("server error: %d", status)
			}
			// 成功
			return nil, nil
		})

		// 如果 cb.Execute 返回错误，说明：
		// 1. 熔断器开启中 (ErrOpenState) -> 直接拦截
		// 2. 或者是内部业务 500 错误 -> 已经被记录
		if err != nil {
			// 区分是熔断拦截还是业务错误
			if errors.Is(err, gobreaker.ErrOpenState) {
				// 熔断保护中
				logger.Warn("请求被熔断器拦截", zap.String("path", c.Request.URL.Path))
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
					"code": 7,
					"msg":  response.LocalizeMessage(c, "服务繁忙，请稍后重试 (Breaker Open)"),
				})
				return
			}

			// 如果是业务本身的 500，Gin 已经处理了响应，这里只需记录
		}
	}
}
