package observability

import (
	"context"
	"github.com/CIPFZ/video-cms/internal/core/config"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// createResource 创建 otel resource
func createResource(cfg config.Observability) (*resource.Resource, error) {
	return resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
}
