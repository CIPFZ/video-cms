package observability

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/CIPFZ/video-cms/internal/core/config"
	corelog "github.com/CIPFZ/video-cms/internal/core/log"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	logspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/proto"
)

func TestOTLPHTTPExportsAllSignalsAndCorrelatesLogs(t *testing.T) {
	var mu sync.Mutex
	received := map[string][]byte{}
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		received[r.URL.Path] = body
		mu.Unlock()
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.WriteHeader(200)
	}))
	defer receiver.Close()
	cfg := config.Observability{Exporter: "otel", ServiceName: "cms-test", TraceSampleRatio: 1, OtelExporter: config.OtelExporterConfig{Protocol: "http", Endpoint: strings.TrimPrefix(receiver.URL, "http://"), Insecure: true}}
	previousTracer, previousMeter, previousPropagator := otel.GetTracerProvider(), otel.GetMeterProvider(), otel.GetTextMapPropagator()
	defer func() {
		otel.SetTracerProvider(previousTracer)
		otel.SetMeterProvider(previousMeter)
		otel.SetTextMapPropagator(previousPropagator)
	}()
	closeTraces, err := InitTraces(cfg)
	require.NoError(t, err)
	closeMetrics, err := InitMetrics(cfg)
	require.NoError(t, err)
	provider, closeLogs, err := InitLogs(cfg)
	require.NoError(t, err)
	ctx := otel.GetTextMapPropagator().Extract(context.Background(), propagation.MapCarrier{"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"})
	ctx, span := otel.Tracer("test").Start(ctx, "operation")
	require.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", trace.SpanContextFromContext(ctx).TraceID().String())
	logger := zap.New(corelog.NewOTELLogCore(&config.OTELLoggerConfig{LogProvider: provider}, zapcore.InfoLevel))
	logger.Info("operation", zap.Any("context", ctx), zap.Float64("duration", 1.25))
	counter, err := otel.Meter("test").Int64Counter("cms.test.operations")
	require.NoError(t, err)
	counter.Add(ctx, 1)
	span.End()
	require.NoError(t, closeLogs(context.Background()))
	require.NoError(t, closeMetrics(context.Background()))
	require.NoError(t, closeTraces(context.Background()))
	mu.Lock()
	defer mu.Unlock()
	for _, path := range []string{"/v1/traces", "/v1/metrics", "/v1/logs"} {
		require.NotEmpty(t, received[path], path)
	}
	var exported logspb.ExportLogsServiceRequest
	require.NoError(t, proto.Unmarshal(received["/v1/logs"], &exported))
	require.Len(t, exported.ResourceLogs, 1)
	require.Len(t, exported.ResourceLogs[0].ScopeLogs, 1)
	records := exported.ResourceLogs[0].ScopeLogs[0].LogRecords
	require.Len(t, records, 1)
	require.Equal(t, "operation", records[0].Body.GetStringValue())
	require.Len(t, records[0].TraceId, 16)
	found := false
	for _, attr := range records[0].Attributes {
		if attr.Key == "duration" {
			found = true
			require.Equal(t, 1.25, attr.Value.GetDoubleValue())
		}
	}
	require.True(t, found)
}
