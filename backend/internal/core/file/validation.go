package file

import (
	"errors"
	"fmt"
	"mime/multipart"
	"path"
	"strings"

	"github.com/CIPFZ/video-cms/internal/core/config"
)

func SanitizeUploadName(name string) string {
	normalized := strings.ReplaceAll(strings.TrimSpace(name), "\\", "/")
	base := path.Base(normalized)
	base = strings.TrimSpace(base)
	if base == "" || base == "." || base == "/" {
		return "file"
	}
	return base
}

func ValidateUpload(cfg config.FileConfig, file *multipart.FileHeader) error {
	if file == nil {
		return errors.New("file is required")
	}

	name := SanitizeUploadName(file.Filename)
	ext := strings.ToLower(path.Ext(name))
	if len(cfg.AllowExt) > 0 {
		allowed := false
		for _, candidate := range cfg.AllowExt {
			if strings.ToLower(strings.TrimSpace(candidate)) == ext {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("file extension %s is not allowed", ext)
		}
	}

	if cfg.MaxMb > 0 {
		maxBytes := cfg.MaxMb * 1024 * 1024
		if file.Size > maxBytes {
			return fmt.Errorf("file size %d exceeds limit %d", file.Size, maxBytes)
		}
	}

	return nil
}
