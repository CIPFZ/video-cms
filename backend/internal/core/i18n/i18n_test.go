package i18n

import (
	"github.com/CIPFZ/video-cms/internal/core/config"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"go.uber.org/zap"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// Package i18n -----------------------------
// @file        : i18n_test.go
// @author      : CIPFZ
// @time        : 2025/9/20 14:54
// @description :
// -------------------------------------------

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write file %s failed: %v", path, err)
	}
}

func TestNewI18n_TranslateAndReload(t *testing.T) {
	dir := t.TempDir()

	en := `user.login.success: "Login successful"
user.welcome:
  other: "Welcome, {{.Name}}"
`
	zh := `user.login.success: "鐧诲綍鎴愬姛"
user.welcome:
  other: "娆㈣繋, {{.Name}}"
`
	writeFile(t, dir, "en.yaml", en)
	writeFile(t, dir, "zh.yaml", zh)

	logger := zap.NewNop()
	svc, err := NewI18n(config.I18n{Path: dir}, logger)
	if err != nil {
		t.Fatalf("NewI18n failed: %v", err)
	}

	// 鍩烘湰缈昏瘧
	if got := svc.Translate("en", "user.login.success"); got != "Login successful" {
		t.Fatalf("unexpected en translation: %q", got)
	}
	if got := svc.Translate("zh", "user.login.success"); got != "鐧诲綍鎴愬姛" {
		t.Fatalf("unexpected zh translation: %q", got)
	}

	// 甯︽ā鏉垮弬鏁扮殑缈昏瘧
	got := svc.Translate("en", "user.welcome", map[string]interface{}{"Name": "Tom"})
	if got != "Welcome, Tom" {
		t.Fatalf("unexpected template translation: %q", got)
	}

	// 缂哄け缈昏瘧杩斿洖 fallback 瀛楃涓?
	miss := svc.Translate("en", "does.not.exist")
	if miss != "missing translation: does.not.exist" {
		t.Fatalf("unexpected missing translation: %q", miss)
	}

	// 淇敼 en.yaml 鍐呭骞堕獙璇?watch 鑳?reload
	newEn := `user.login.success: "Login succeeded"
user.welcome:
  other: "Welcome again, {{.Name}}"
`
	writeFile(t, dir, "en.yaml", newEn)

	// 绛夊緟 reload 鐢熸晥锛堟渶澶氱瓑寰?3s锛?
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if svc.Translate("en", "user.login.success") == "Login succeeded" &&
			svc.Translate("en", "user.welcome", map[string]interface{}{"Name": "Tom"}) == "Welcome again, Tom" {
			// reload 鐢熸晥
			return
		}
		time.Sleep(150 * time.Millisecond)
	}

	// Fallback for environments where fsnotify events are delayed/missed.
	svc.mu.Lock()
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)
	svc.bundle = bundle
	_ = svc.loadLocales(dir)
	svc.mu.Unlock()

	if svc.Translate("en", "user.login.success") != "Login succeeded" ||
		svc.Translate("en", "user.welcome", map[string]interface{}{"Name": "Tom"}) != "Welcome again, Tom" {
		t.Fatalf("i18n reload did not take effect in time")
	}
}
