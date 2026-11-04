package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/CIPFZ/video-cms/internal/svc"
	"github.com/gin-gonic/gin"
)

// Health routes bypass business throttling, circuit breaking and CORS policies.
func registerHealthRoutes(r *gin.Engine, svcCtx *svc.ServiceContext) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().Unix()})
	})
	checks := map[string]func(context.Context) error{
		"database": func(ctx context.Context) error {
			if svcCtx.DB == nil {
				return fmt.Errorf("not initialized")
			}
			db, err := svcCtx.DB.DB()
			if err != nil {
				return err
			}
			return db.PingContext(ctx)
		},
	}
	if svcCtx.Config.System.UseRedis {
		checks["redis"] = func(ctx context.Context) error {
			if svcCtx.Redis == nil {
				return fmt.Errorf("not initialized")
			}
			return svcCtx.Redis.Ping(ctx).Err()
		}
	}
	if svcCtx.Config.System.UseMongo {
		checks["mongo"] = func(ctx context.Context) error {
			if svcCtx.Mongo == nil {
				return fmt.Errorf("not initialized")
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			deadline, _ := ctx.Deadline()
			return svcCtx.Mongo.Ping(int64(time.Until(deadline)))
		}
	}
	r.GET("/ready", readinessHandler(checks))
}

func readinessHandler(checks map[string]func(context.Context) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		status := http.StatusOK
		result := make(map[string]string, len(checks))
		for name, check := range checks {
			result[name] = "ok"
			if check(ctx) != nil {
				status = http.StatusServiceUnavailable
				result[name] = "unavailable"
			}
		}
		state := "ok"
		if status != http.StatusOK {
			state = "unavailable"
		}
		// Never expose DSNs, credentials or raw dependency errors on a public route.
		c.JSON(status, gin.H{"status": state, "checks": result})
	}
}
