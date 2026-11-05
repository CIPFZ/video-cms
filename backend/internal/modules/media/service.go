package media

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Service struct {
	db     *gorm.DB
	client *http.Client
	cancel context.CancelFunc
	done   chan struct{}
	once   sync.Once

	facetMu  sync.Mutex
	facets   *PublicFacets
	facetsAt time.Time
}
type Config struct{ DB *gorm.DB }

func NewService(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, errors.New("media collector requires a database")
	}
	return &Service{db: cfg.DB, client: safeClient(), done: make(chan struct{})}, nil
}
func (s *Service) ListSources(ctx context.Context) ([]Source, error) {
	var rows []Source
	err := s.db.WithContext(ctx).Order("priority, id").Find(&rows).Error
	return rows, err
}
func (s *Service) GetSource(ctx context.Context, id uint) (Source, error) {
	var row Source
	err := s.db.WithContext(ctx).First(&row, id).Error
	return row, err
}
func defaults(item *Source) {
	item.Name = strings.TrimSpace(item.Name)
	item.APIURL = strings.TrimSpace(item.APIURL)
	if item.APIType == "" {
		item.APIType = "json"
	}
	if item.TimeoutSeconds == 0 {
		item.TimeoutSeconds = 15
	}
	if item.RequestIntervalSeconds == 0 {
		item.RequestIntervalSeconds = 3
	}
	if item.Priority == 0 {
		item.Priority = 100
	}
}
func validateSource(item Source) error {
	if item.Name == "" || len([]rune(item.Name)) > 100 {
		return errors.New("来源名称须为 1–100 字")
	}
	if err := validateURL(item.APIURL); err != nil {
		return err
	}
	if item.APIType != "" && item.APIType != "json" {
		return errors.New("第一阶段只支持 JSON 接口")
	}
	if item.TimeoutSeconds < 0 || item.TimeoutSeconds > 120 || item.RequestIntervalSeconds < 0 || item.RequestIntervalSeconds > 60 {
		return errors.New("超时或请求间隔超出范围")
	}
	return nil
}
func (s *Service) CreateSource(ctx context.Context, item Source) (Source, error) {
	defaults(&item)
	if err := validateSource(item); err != nil {
		return item, err
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Select all fields so GORM does not turn an explicitly disabled source on.
		enabled := item.Enabled
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		if !enabled {
			if err := tx.Model(&item).Update("enabled", false).Error; err != nil {
				return err
			}
			item.Enabled = false
		}
		return seedSchedules(tx, item.ID)
	})
	return item, err
}
func (s *Service) UpdateSource(ctx context.Context, item Source) (Source, error) {
	defaults(&item)
	if err := validateSource(item); err != nil {
		return item, err
	}
	old, err := s.GetSource(ctx, item.ID)
	if err != nil {
		return item, err
	}
	if old.LeaseUntil != nil && old.LeaseUntil.After(time.Now()) {
		return item, errors.New("来源正在执行任务，请先暂停任务")
	}
	err = s.db.WithContext(ctx).Model(&old).Select("name", "api_url", "api_type", "enabled", "priority", "allow_categories", "deny_categories", "request_interval_seconds", "timeout_seconds", "full_enabled").Updates(&item).Error
	if err != nil {
		return item, err
	}
	return s.GetSource(ctx, item.ID)
}

type ItemDetail struct {
	Item      Item             `json:"item"`
	Playbacks []SourcePlayback `json:"playbacks"`
}
type SourcePlayback struct {
	Source     Source     `json:"source"`
	SourceItem SourceItem `json:"sourceItem"`
	Lines      []PlayLine `json:"lines"`
}

func (s *Service) GetItemDetail(ctx context.Context, id uint) (ItemDetail, error) {
	result := ItemDetail{Playbacks: []SourcePlayback{}}
	if err := s.db.WithContext(ctx).First(&result.Item, id).Error; err != nil {
		return result, err
	}
	var links []SourceItem
	if err := s.db.WithContext(ctx).Where("cms_media_id = ? AND state = ?", id, "active").Find(&links).Error; err != nil {
		return result, err
	}
	for _, link := range links {
		src, err := s.GetSource(ctx, link.SourceID)
		if err != nil {
			return result, err
		}
		if !src.Enabled || src.Suspended {
			continue
		}
		lines := []PlayLine{}
		if err := s.db.WithContext(ctx).Where("source_item_id = ? AND enabled = ?", link.ID, true).Order("sort_order, id").Preload("Episodes", func(db *gorm.DB) *gorm.DB { return db.Order("episode_no") }).Find(&lines).Error; err != nil {
			return result, err
		}
		for i := range lines {
			lines[i].SourceName = src.Name
		}
		result.Playbacks = append(result.Playbacks, SourcePlayback{src, link, lines})
	}
	return result, nil
}
func (s *Service) ListItems(ctx context.Context, page, size int, query, kind string) (map[string]any, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	var rows []Item
	var total int64
	db := s.db.WithContext(ctx).Model(&Item{})
	if query != "" {
		pattern := "%" + query + "%"
		db = db.Where("title LIKE ? OR original_title LIKE ? OR genres LIKE ?", pattern, pattern, pattern)
	}
	if kind != "" {
		// Media type values may contain multiple canonical tags (for example movie,anime).
		// Match a selected tag without requiring the UI to know the storage delimiter.
		db = db.Where("media_type = ? OR media_type LIKE ? OR media_type LIKE ? OR media_type LIKE ?", kind, kind+",%", "%,"+kind+",%", "%,"+kind)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	err := db.Order("updated_at desc,id desc").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return map[string]any{"list": rows, "total": total, "page": page, "pageSize": size}, err
}

type RunFilter struct {
	SourceID uint
	Mode     string // a concrete collection mode; empty or "collect" for all
	Status   string
	Page     int
	PageSize int
}

// ListRuns filters and paginates collection runs in the database. Health checks
// have their own history; runs queued as health checks by older versions stay hidden.
func (s *Service) ListRuns(ctx context.Context, f RunFilter) (map[string]any, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 100 {
		f.PageSize = 10
	}
	db := s.db.WithContext(ctx).Model(&CollectionRun{}).Where("mode <> ?", "health")
	if f.SourceID > 0 {
		db = db.Where("source_id = ?", f.SourceID)
	}
	if f.Mode != "" && f.Mode != "collect" {
		db = db.Where("mode = ?", f.Mode)
	}
	// Summary counts ignore the status filter so the cards keep their meaning.
	var groups []struct {
		Status string
		Count  int64
	}
	if err := db.Session(&gorm.Session{}).Select("status, COUNT(*) AS count").Group("status").Scan(&groups).Error; err != nil {
		return nil, err
	}
	summary := map[string]int64{}
	for _, g := range groups {
		summary[g.Status] = g.Count
	}
	if f.Status != "" {
		db = db.Where("status = ?", f.Status)
	}
	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []CollectionRun
	err := db.Preload("PageFailures", func(db *gorm.DB) *gorm.DB {
		return db.Order("page, id")
	}).Order("id desc").Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).Find(&rows).Error
	return map[string]any{"list": rows, "total": total, "page": f.Page, "pageSize": f.PageSize, "summary": summary}, err
}

// ListRunItems pages the titles a run inserted, updated or failed to save.
func (s *Service) ListRunItems(ctx context.Context, runID uint, action string, page, size int) (map[string]any, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	db := s.db.WithContext(ctx).Model(&CollectionRunItem{}).Where("collection_run_id = ?", runID)
	var groups []struct {
		Action string
		Count  int64
	}
	if err := db.Session(&gorm.Session{}).Select("action, COUNT(*) AS count").Group("action").Scan(&groups).Error; err != nil {
		return nil, err
	}
	summary := map[string]int64{}
	for _, g := range groups {
		summary[g.Action] = g.Count
	}
	if action != "" {
		db = db.Where("action = ?", action)
	}
	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []CollectionRunItem
	err := db.Order("id").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return map[string]any{"list": rows, "total": total, "page": page, "pageSize": size, "summary": summary}, err
}
func (s *Service) ListHealth(ctx context.Context, id uint) ([]HealthCheck, error) {
	var rows []HealthCheck
	err := s.db.WithContext(ctx).Where("source_id = ?", id).Order("id desc").Limit(50).Find(&rows).Error
	return rows, err
}
