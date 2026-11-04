package file

import (
	"github.com/CIPFZ/video-cms/internal/core/config"
	"go.uber.org/zap"
)

func NewFileService(cfg config.FileConfig, logger *zap.Logger) OSS {
	switch cfg.Driver {
	case "local":
		return NewLocalDriver(cfg.Local, logger)
	case "minio":
		// ✨ 注册 MinIO
		return NewMinioDriver(cfg.Minio, logger)
	default:
		return NewLocalDriver(cfg.Local, logger)
	}
}
