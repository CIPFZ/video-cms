package file

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/CIPFZ/video-cms/internal/core/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type LocalDriver struct {
	config config.LocalConfig
	logger *zap.Logger
	tracer trace.Tracer
}

func NewLocalDriver(cfg config.LocalConfig, logger *zap.Logger) *LocalDriver {
	// 启动时确保根目录存在
	if err := os.MkdirAll(cfg.Path, os.ModePerm); err != nil {
		logger.Error("failed to create local upload root directory", zap.Error(err), zap.String("path", cfg.Path))
	}
	return &LocalDriver{
		config: cfg,
		logger: logger,
		tracer: otel.Tracer("core.file.local"),
	}
}

// Upload 本地上传
// 接收 ctx 用于链路追踪串联
func (l *LocalDriver) Upload(ctx context.Context, file *multipart.FileHeader, fileName string) (string, string, error) {
	// ✨ 1. 开启 Span
	// 父 Span (Context) 通常来自 Service -> API -> HTTP Request
	ctx, span := l.tracer.Start(ctx, "LocalFileSystem.Upload", trace.WithAttributes(
		attribute.String("db.system", "filesystem"), // 标识系统类型
		attribute.String("file.name", fileName),
		attribute.Int64("file.size", file.Size),
	))
	defer span.End()

	// 2. 打开源文件
	src, err := file.Open()
	if err != nil {
		recordError(span, err)
		return "", "", err
	}
	defer src.Close()

	// 3. 计算路径
	// 建议按日期分目录，避免单文件夹下文件过多影响 I/O 性能
	// e.g. "2025-12-07"
	subDir := time.Now().Format("2006-01-02")

	// 物理存储绝对路径 (使用 filepath 兼容 Windows/Linux)
	// e.g. /data/uploads/2025-12-07
	storeDir := filepath.Join(l.config.Path, subDir)

	// 创建目录
	if err := os.MkdirAll(storeDir, os.ModePerm); err != nil {
		recordError(span, err)
		return "", "", fmt.Errorf("failed to create dir: %w", err)
	}

	// 完整物理文件路径
	// e.g. /data/uploads/2025-12-07/uuid.png
	fullPath := filepath.Join(storeDir, fileName)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		recordError(span, err)
		return "", "", fmt.Errorf("failed to create upload directory: %w", err)
	}

	// 记录到 Trace
	span.SetAttributes(attribute.String("file.path", fullPath))

	// 4. 创建目标文件
	out, err := os.Create(fullPath)
	if err != nil {
		recordError(span, err)
		return "", "", fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// 5. 写入内容
	written, err := io.Copy(out, src)
	if err != nil {
		recordError(span, err)
		return "", "", fmt.Errorf("failed to write file: %w", err)
	}
	span.SetAttributes(attribute.Int64("file.written_bytes", written))

	// 6. 生成访问 URL
	// URL 路径必须使用 '/' (正斜杠)，所以使用 path 包而不是 filepath 包
	// e.g. /uploads/file/2025-12-07/uuid.png
	accessUrl := path.Join(l.config.StorePath, subDir, fileName)

	return accessUrl, fullPath, nil
}

// Delete 本地删除
func (l *LocalDriver) Delete(ctx context.Context, key string) error {
	// ✨ 1. 开启 Span
	_, span := l.tracer.Start(ctx, "LocalFileSystem.Delete", trace.WithAttributes(
		attribute.String("db.system", "filesystem"),
		attribute.String("file.path", key),
	))
	defer span.End()

	// 2. 安全检查：防止路径穿越 (e.g. ../../../etc/passwd)
	// key 在这里就是文件的物理绝对路径
	if strings.Contains(key, "..") {
		err := errors.New("illegal file path: path traversal detected")
		recordError(span, err)
		return err
	}

	// 3. 执行删除
	if err := os.Remove(key); err != nil {
		// 如果文件本身就不存在，通常不视为错误
		if os.IsNotExist(err) {
			span.AddEvent("file not found, skipping delete")
			return nil
		}
		recordError(span, err)
		return err
	}

	return nil
}

// recordError 辅助函数：记录错误到 Span 并设置状态
func recordError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
