package main

import (
	"context"
	"errors"
	"flag"
	"github.com/CIPFZ/video-cms/internal/bootstrap"
	corelog "github.com/CIPFZ/video-cms/internal/core/log"
	"github.com/CIPFZ/video-cms/internal/core/server"
	"github.com/CIPFZ/video-cms/internal/svc"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// ---------------- 1. 初始化系统 ----------------
	configPath := flag.String("f", defaultConfigPath, "config file path")
	flag.Parse()

	serviceCtx := svc.NewServiceContext()

	// 初始化并获取关停函数列表
	allShutdowns, err := bootstrap.Initialize(*configPath, serviceCtx)
	if err != nil {
		log.Fatalf("❌ 系统初始化失败: %v", err)
	}

	// ✨ 确保 Logger 最后刷盘 (stdout/file)
	defer corelog.Sync(serviceCtx.Logger)

	// ---------------- 2. 启动 HTTP 服务 ----------------
	// server.NewServer 内部会调用 InitRouters 进行依赖注入和路由注册
	serviceCtx.SRV = server.NewServer(serviceCtx)
	server.PrintBanner(serviceCtx.Config.System.Port)

	serverErr := make(chan error, 1)
	go func() {
		serviceCtx.Logger.Info("🚀 HTTP服务启动中...", zap.String("addr", serviceCtx.SRV.Addr))
		if listenErr := serviceCtx.SRV.ListenAndServe(); listenErr != nil && !errors.Is(listenErr, http.ErrServerClosed) {
			serverErr <- listenErr
		}
	}()

	// ---------------- 3. 阻塞并等待退出 ----------------
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		serviceCtx.Logger.Error("HTTP service stopped", zap.Error(err))
	case s := <-quit:
		serviceCtx.Logger.Info("🛑 收到退出信号，准备关闭服务...", zap.String("signal", s.String()))
	}

	// ---------------- 4. 优雅退出 ----------------
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// 1. 关停 HTTP 服务 (停止接收新请求)
	if shutdownErr := server.ShutdownServer(shutdownCtx, serviceCtx); shutdownErr != nil {
		serviceCtx.Logger.Warn("HTTP服务关闭异常", zap.Error(shutdownErr))
	}

	// 2. 执行组件关停 (DB, Mongo, Redis, Otel Traces/Metrics, Audit)
	// Close in reverse initialization order: producers before their dependencies.
	for i := len(allShutdowns) - 1; i >= 0; i-- {
		if err := allShutdowns[i](shutdownCtx); err != nil {
			serviceCtx.Logger.Warn("组件关停异常", zap.Error(err))
		}
	}

	// (defer logger.Sync 会在这里执行)
	log.Println("👋 服务已成功关闭")
}

const defaultConfigPath = "./configs/config.yaml"
