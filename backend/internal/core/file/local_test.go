package file

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/CIPFZ/video-cms/internal/core/config"
	"go.uber.org/zap"
)

func TestLocalUploadCreatesUserDirectory(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "avatar.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("image-content")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/upload", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	stream, header, err := request.FormFile("file")
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	defer request.MultipartForm.RemoveAll()

	driver := NewLocalDriver(config.LocalConfig{Path: t.TempDir(), StorePath: "/uploads/file"}, zap.NewNop())
	url, key, err := driver.Upload(context.Background(), header, "user-id/avatar.png")
	if err != nil {
		t.Fatalf("upload under user directory: %v", err)
	}
	content, err := os.ReadFile(key)
	if err != nil || string(content) != "image-content" {
		t.Fatalf("stored upload = %q, error = %v", content, err)
	}
	if !strings.HasPrefix(url, "/uploads/file/") || !strings.HasSuffix(url, "/user-id/avatar.png") {
		t.Fatalf("unexpected upload URL: %s", url)
	}
}
