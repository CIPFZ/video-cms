package middleware

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/CIPFZ/video-cms/internal/core/audit"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/CIPFZ/video-cms/internal/svc"
	"github.com/CIPFZ/video-cms/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

// Capture at most one extra byte so oversized bodies can be omitted altogether.
// Returning len(p) lets TeeReader stream the original body without changing it.
type boundedCapture struct{ bytes.Buffer }

func (b *boundedCapture) Write(p []byte) (int, error) {
	n := len(p)
	remaining := audit.MaxBodySize + 1 - b.Len()
	if remaining > 0 {
		_, _ = b.Buffer.Write(p[:min(remaining, len(p))])
	}
	return n, nil
}

type capturedRequest struct {
	io.Reader
	io.Closer
}

func OperationRecord(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		if svcCtx.AuditRecorder == nil || c.Request.Method == http.MethodGet || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		var requestBody boundedCapture
		mediaType, _, _ := mime.ParseMediaType(c.GetHeader("Content-Type"))
		isJSON := mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
		if c.Request.Body != nil && isJSON {
			original := c.Request.Body
			c.Request.Body = &capturedRequest{Reader: io.TeeReader(original, &requestBody), Closer: original}
		}
		writer := &responseBodyWriter{ResponseWriter: c.Writer}
		c.Writer = writer
		start := time.Now()
		c.Next()

		var traceID, spanID string
		sc := trace.SpanContextFromContext(c.Request.Context())
		if sc.IsValid() {
			traceID, spanID = sc.TraceID().String(), sc.SpanID().String()
		}
		errorSummary := ""
		if len(c.Errors) > 0 {
			errorSummary = fmt.Sprintf("%d handler error(s)", len(c.Errors))
		}
		svcCtx.AuditRecorder.Push(model.SysOperationLog{
			Ip: c.ClientIP(), Method: c.Request.Method,
			Path:  truncateString(c.Request.URL.Path, 2048),
			Agent: truncateString(c.Request.UserAgent(), 512),
			Body:  audit.SanitizeBody(requestBody.Bytes()), Resp: audit.SanitizeBody(writer.body.Bytes()),
			Status: c.Writer.Status(), Latency: time.Since(start),
			UserID: utils.GetUserID(c), TraceID: traceID, SpanID: spanID, ErrorMsg: errorSummary,
		})
	}
}

func truncateString(s string, max int) string {
	if len(s) > max {
		return strings.ToValidUTF8(s[:max], "")
	}
	return s
}

type responseBodyWriter struct {
	gin.ResponseWriter
	body boundedCapture
}

func (r *responseBodyWriter) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	_, _ = r.body.Write(b[:n])
	return n, err
}

func (r *responseBodyWriter) WriteString(s string) (int, error) {
	return r.Write([]byte(s))
}
