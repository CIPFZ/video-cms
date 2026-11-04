package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/CIPFZ/video-cms/internal/svc"

	"go.uber.org/zap"
)

// NewServer 创建 HTTP 服务
func NewServer(serviceCtx *svc.ServiceContext) *http.Server {
	// 初始化路由
	engine := InitRouters(serviceCtx)
	address := fmt.Sprintf(":%d", serviceCtx.Config.System.Port)

	return &http.Server{
		Addr:              address,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadTimeout:       10 * time.Minute,
		WriteTimeout:      10 * time.Minute,
		MaxHeaderBytes:    1 << 20,
	}
}

// ShutdownServer 优雅关闭 HTTP 服务
func ShutdownServer(ctx context.Context, serviceCtx *svc.ServiceContext) error {
	serviceCtx.Logger.Info("正在关闭 HTTP 服务...")

	// 检查 SRV 是否为 nil
	if serviceCtx.SRV == nil {
		serviceCtx.Logger.Warn("HTTP 服务实例(SRV)为 nil，无需关闭。")
		return nil
	}

	if err := serviceCtx.SRV.Shutdown(ctx); err != nil {
		serviceCtx.Logger.Error("HTTP服务关闭异常", zap.Error(err))
		return err
	}
	serviceCtx.Logger.Info("HTTP 服务已优雅退出 ✅")
	return nil
}
