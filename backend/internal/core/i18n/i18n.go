package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/CIPFZ/video-cms/internal/core/config"

	"github.com/fsnotify/fsnotify"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"go.uber.org/zap"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// Package i18n -----------------------------
// @file        : i18n.go
// @author      : CIPFZ
// @time        : 2025/9/19 17:09
// @description :
// -------------------------------------------

type Service struct {
	mu     sync.RWMutex
	bundle *i18n.Bundle
	path   string
	logger *zap.Logger
}

// NewI18n 初始化 I18n 服务
func NewI18n(config config.I18n, logger *zap.Logger) (*Service, error) {
	// 初始化 bundle
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)

	s := &Service{
		bundle: bundle,
		path:   config.Path,
		logger: logger,
	}

	// 先加载一次
	if err := s.loadLocales(config.Path); err != nil {
		return nil, err
	}

	// 开启文件监听
	go s.watch()

	return s, nil
}

// loadLocales 加载翻译文件到 bundle
func (s *Service) loadLocales(localePath string) error {
	files, err := os.ReadDir(localePath)
	if err != nil {
		return fmt.Errorf("failed to read locale dir: %w", err)
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		path := filepath.Join(localePath, f.Name())
		if _, err := s.bundle.LoadMessageFile(path); err != nil {
			s.logger.Error("failed to load locale file",
				zap.String("file", path),
				zap.Error(err),
			)
			continue
		}
		s.logger.Info("loaded locale file", zap.String("file", path))
	}
	return nil
}

// watch 监听目录变化，动态刷新翻译
func (s *Service) watch() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		s.logger.Error("failed to start fsnotify watcher", zap.Error(err))
		return
	}
	defer watcher.Close()

	if err := watcher.Add(s.path); err != nil {
		s.logger.Error("failed to watch locale path", zap.String("path", s.path), zap.Error(err))
		return
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				s.logger.Info("locale file changed, reloading", zap.String("file", event.Name))
				s.mu.Lock()
				// 每次更新时新建 bundle，防止重复 key 覆盖
				bundle := i18n.NewBundle(language.English)
				bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)
				s.bundle = bundle
				_ = s.loadLocales(s.path)
				s.mu.Unlock()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			s.logger.Error("fsnotify error", zap.Error(err))
		}
	}
}

// Translate 翻译方法（支持可选参数）
func (s *Service) Translate(lang, messageID string, templateData ...map[string]interface{}) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var data map[string]interface{}
	if len(templateData) > 0 {
		data = templateData[0]
	}

	localizer := i18n.NewLocalizer(s.bundle, lang)
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: data,
	})
	if err != nil {
		s.logger.Warn("missing translation",
			zap.String("lang", lang),
			zap.String("messageID", messageID),
			zap.Error(err),
		)
		return fmt.Sprintf("missing translation: %s", messageID)
	}
	return msg
}

// Lookup lets the HTTP boundary distinguish an unknown internal error from a
// translated business message without returning translation keys to clients.
func (s *Service) Lookup(lang, id string, data map[string]interface{}) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, err := i18n.NewLocalizer(s.bundle, lang).Localize(&i18n.LocalizeConfig{MessageID: id, TemplateData: data})
	return value, err == nil
}
