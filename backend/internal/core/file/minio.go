package file

import (
	"context"
	"fmt"
	"mime/multipart"
	"path"

	"github.com/CIPFZ/video-cms/internal/core/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type MinioDriver struct {
	client *minio.Client
	config config.MinioConfig
	logger *zap.Logger
	tracer trace.Tracer // ✨ 添加 tracer
}

func NewMinioDriver(cfg config.MinioConfig, logger *zap.Logger) *MinioDriver {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		logger.Panic("minio init failed", zap.Error(err))
	}

	driver := &MinioDriver{
		client: client,
		config: cfg,
		logger: logger,
		// ✨ 初始化 tracer
		tracer: otel.Tracer("core.file.minio"),
	}

	driver.ensureBucket()
	return driver
}

func (m *MinioDriver) ensureBucket() {
	// Bucket 初始化通常在启动时做，用 Background context
	ctx := context.Background()

	// 这里也可以加个 Span 记录初始化过程
	ctx, span := m.tracer.Start(ctx, "Minio.EnsureBucket")
	defer span.End()

	bucketName := m.config.Bucket
	// 1. 检查 Bucket 是否存在
	exists, err := m.client.BucketExists(ctx, bucketName)
	if err != nil {
		m.logger.Error("check bucket exists failed", zap.Error(err))
		// 如果连检查都失败，后面就别跑了
		return
	}

	// 2. 如果不存在，创建它
	if !exists {
		if err := m.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
			m.logger.Error("create bucket failed", zap.Error(err))
			return
		}
		m.logger.Info("minio bucket created successfully", zap.String("bucket", bucketName))
	}

	// ✨✨✨ 3. 关键修改：将设置 Policy 的代码移到 if !exists 外面 ✨✨✨
	// 强制设置为公开读 (Public Read)
	// 这样每次服务重启，都会确保权限是正确的，防止人为误操作关掉权限
	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {"AWS": ["*"]},
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::%s/*"]
			}
		]
	}`, bucketName)
	if err := m.client.SetBucketPolicy(ctx, bucketName, policy); err != nil {
		m.logger.Error("set bucket policy failed", zap.Error(err))
	}
}

// Upload 上传
func (m *MinioDriver) Upload(ctx context.Context, file *multipart.FileHeader, fileName string) (string, string, error) {
	// ✨ 1. 开始 Span
	// ctx 会自动继承上游 (HTTP Request -> Service) 的 TraceID
	ctx, span := m.tracer.Start(ctx, "Minio.PutObject", trace.WithAttributes(
		attribute.String("db.system", "minio"), // 标准属性
		attribute.String("s3.bucket", m.config.Bucket),
		attribute.String("s3.key", fileName),
		attribute.Int64("file.size", file.Size),
	))
	defer span.End()

	src, err := file.Open()
	if err != nil {
		span.RecordError(err)
		return "", "", err
	}
	defer src.Close()

	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// ✨ 2. 调用 MinIO (传入 ctx)
	// MinIO SDK 会处理 context 取消等逻辑，但它本身不会自动产生 Trace
	info, err := m.client.PutObject(ctx, m.config.Bucket, fileName, src, file.Size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		// 记录错误到 Trace
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", "", err
	}

	// 记录成功属性
	span.SetAttributes(attribute.String("minio.etag", info.ETag))

	// ... (拼接 URL 逻辑保持不变)
	accessUrl := path.Join(m.config.PreviewBasePath, fileName)
	if m.config.PreviewBasePath != "" && m.config.PreviewBasePath[len(m.config.PreviewBasePath)-1] != '/' {
		accessUrl = m.config.PreviewBasePath + "/" + fileName
	}

	return accessUrl, fileName, nil
}

// Delete 删除
func (m *MinioDriver) Delete(ctx context.Context, key string) error {
	// ✨ 1. 开始 Span
	ctx, span := m.tracer.Start(ctx, "Minio.RemoveObject", trace.WithAttributes(
		attribute.String("db.system", "minio"),
		attribute.String("s3.bucket", m.config.Bucket),
		attribute.String("s3.key", key),
	))
	defer span.End()

	// ✨ 2. 调用 MinIO
	err := m.client.RemoveObject(ctx, m.config.Bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}
