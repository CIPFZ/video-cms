package middleware

import (
	"context"
	"github.com/CIPFZ/video-cms/pkg/utils"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// GinLoggerMiddleware 返回一个 Gin 中间件：把 trace_id, span_id 注入 logger 和 context
func GinLoggerMiddleware(base *zap.Logger) gin.HandlerFunc {
	if base == nil {
		base = zap.NewNop()
	}
	return func(c *gin.Context) {
		// 从 request context 获取 SpanContext
		spanCtx := trace.SpanContextFromContext(c.Request.Context())
		loggerWithTrace := base
		if spanCtx.HasTraceID() && spanCtx.HasSpanID() {
			loggerWithTrace = base.With(
				zap.String("trace_id", spanCtx.TraceID().String()),
				zap.String("span_id", spanCtx.SpanID().String()),
				zap.String("http.method", c.Request.Method),
				zap.String("http.path", c.Request.URL.Path),
			)
		}
		loggerWithTrace = loggerWithTrace.With(zap.Any("context", c.Request.Context()))
		// 存入 gin.Context，后续 handler 可以用
		c.Set(utils.LoggerKey, loggerWithTrace)

		// 替换 request.Context()，让下游也能取到
		newCtx := context.WithValue(c.Request.Context(), utils.LoggerKey, loggerWithTrace)
		// 将其赋值回 c.Request
		c.Request = c.Request.WithContext(newCtx)

		c.Next()
	}
}
