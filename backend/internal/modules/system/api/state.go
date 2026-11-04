package api

import (
	"context"
	"github.com/CIPFZ/video-cms/internal/migrations"
	"github.com/CIPFZ/video-cms/internal/svc"
	"github.com/CIPFZ/video-cms/pkg/response"
	"github.com/gin-gonic/gin"
	"runtime"
	"time"
)

type StateApi struct {
	svcCtx  *svc.ServiceContext
	started time.Time
}

func NewStateApi(s *svc.ServiceContext) *StateApi { return &StateApi{svcCtx: s, started: time.Now()} }

// GetServerInfo is a lightweight dependency overview. Historical resource
// metrics belong in the telemetry backend, with an explicit container/host scope.
func (a *StateApi) GetServerInfo(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	checks := map[string]string{"backend": "ok", "database": "unavailable", "redis": "disabled", "mongo": "disabled"}
	if a.svcCtx.DB != nil {
		if pool, err := a.svcCtx.DB.DB(); err == nil && pool.PingContext(ctx) == nil {
			checks["database"] = "ok"
		}
	}
	if a.svcCtx.Config.System.UseRedis {
		checks["redis"] = "unavailable"
		if a.svcCtx.Redis != nil && a.svcCtx.Redis.Ping(ctx).Err() == nil {
			checks["redis"] = "ok"
		}
	}
	if a.svcCtx.Config.System.UseMongo {
		checks["mongo"] = "unavailable"
		if a.svcCtx.Mongo != nil && a.svcCtx.Mongo.Ping(1000000000) == nil {
			checks["mongo"] = "ok"
		}
	}
	version := a.svcCtx.Config.Observable.ServiceVersion
	if version == "" {
		version = "dev"
	}
	response.OkWithData(gin.H{"server": gin.H{"checks": checks, "version": version, "schemaVersion": migrations.Latest, "goVersion": runtime.Version(), "uptimeSeconds": int(time.Since(a.started).Seconds()), "observabilityEnabled": a.svcCtx.Config.Observable.Exporter == "otel"}}, c)
}
