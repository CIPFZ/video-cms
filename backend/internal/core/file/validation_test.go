package file

import (
	"mime/multipart"
	"net/textproto"
	"testing"

	"github.com/CIPFZ/video-cms/internal/core/config"
)

func TestSanitizeUploadNameRemovesTraversal(t *testing.T) {
	got := SanitizeUploadName(`..\..\evil/test.png`)
	if got != "test.png" {
		t.Fatalf("SanitizeUploadName() = %q, want %q", got, "test.png")
	}
}

func TestValidateUploadRejectsDisallowedExtension(t *testing.T) {
	cfg := config.FileConfig{
		MaxMb:    2,
		AllowExt: []string{".png", ".jpg"},
	}

	err := ValidateUpload(cfg, &multipart.FileHeader{
		Filename: "avatar.exe",
		Size:     128,
		Header:   textproto.MIMEHeader{"Content-Type": {"application/octet-stream"}},
	})
	if err == nil {
		t.Fatal("ValidateUpload() error = nil, want extension rejection")
	}
}

func TestValidateUploadRejectsOversizeFile(t *testing.T) {
	cfg := config.FileConfig{
		MaxMb:    1,
		AllowExt: []string{".png"},
	}

	err := ValidateUpload(cfg, &multipart.FileHeader{
		Filename: "avatar.png",
		Size:     2 * 1024 * 1024,
		Header:   textproto.MIMEHeader{"Content-Type": {"image/png"}},
	})
	if err == nil {
		t.Fatal("ValidateUpload() error = nil, want oversize rejection")
	}
}
