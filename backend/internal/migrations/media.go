package migrations

import (
	"errors"
	"strconv"
	"strings"

	"github.com/CIPFZ/video-cms/internal/modules/media"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func ensureMediaModule(tx *gorm.DB, prefix string) error {
	parent := model.SysMenu{
		Path: "/media", Name: "影视资源", NameEn: "Media resources",
		Component: "components/RouterLayout", Locale: "menu.media", Icon: "VideoCameraOutlined", Sort: 17,
	}
	if err := tx.Where("path = ?", parent.Path).FirstOrCreate(&parent).Error; err != nil {
		return err
	}
	if err := tx.Model(&model.SysMenu{}).Where("path = ?", parent.Path).Updates(map[string]any{
		"component": "components/RouterLayout",
		"name":      parent.Name,
		"name_en":   parent.NameEn,
		"locale":    parent.Locale,
	}).Error; err != nil {
		return err
	}
	var adminCount int64
	if err := tx.Model(&model.SysAuthority{}).Where("authority_id = ?", 1).Count(&adminCount).Error; err != nil {
		return err
	}
	menus := []model.SysMenu{
		{Path: "/media/sources", Name: "资源站点", NameEn: "Media sources", Component: "media/sources", Icon: "DatabaseOutlined", Sort: 1, ParentId: parent.ID},
		{Path: "/media/schedules", Name: "采集计划", NameEn: "Collection schedules", Component: "media/schedules", Icon: "ScheduleOutlined", Sort: 2, ParentId: parent.ID},
		{Path: "/media/runs", Name: "采集日志", NameEn: "Collection logs", Component: "media/runs", Icon: "FileSearchOutlined", Sort: 3, ParentId: parent.ID},
		{Path: "/media/health", Name: "连通性检查", NameEn: "Source health", Component: "media/health", Icon: "ApiOutlined", Sort: 4, ParentId: parent.ID},
		{Path: "/media/library", Name: "影视库", NameEn: "Media library", Component: "media/library", Icon: "AppstoreOutlined", Sort: 5, ParentId: parent.ID},
		{Path: "/media/player", Name: "影视播放", NameEn: "Media player", Component: "media/player", HideInMenu: true, Sort: 99, ParentId: parent.ID},
	}
	for _, desired := range menus {
		entry := desired
		if err := tx.Where("path = ?", desired.Path).FirstOrCreate(&entry).Error; err != nil {
			return err
		}
		if err := tx.Model(&entry).Updates(map[string]any{
			"name": desired.Name, "name_en": desired.NameEn, "component": desired.Component, "locale": desired.Locale,
			"icon": desired.Icon, "sort": desired.Sort, "parent_id": parent.ID, "hide_in_menu": desired.HideInMenu,
		}).Error; err != nil {
			return err
		}
		if adminCount > 0 {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.SysAuthorityMenu{MenuId: entry.ID, AuthorityId: 1}).Error; err != nil {
				return err
			}
		}
	}
	if adminCount > 0 {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.SysAuthorityMenu{MenuId: parent.ID, AuthorityId: 1}).Error; err != nil {
			return err
		}
	}
	endpoints := []struct {
		suffix string
		method string
		desc   string
	}{
		{"/media/sources", "GET", "List media sources"},
		{"/media/sources", "POST", "Create media source"},
		{"/media/sources/:id", "PUT", "Update media source"},
		{"/media/sources/:id/health", "POST", "Check media source health"},
		{"/media/sources/:id/collect", "POST", "Collect media source records"},
		{"/media/items", "GET", "List collected media items"},
		{"/media/items/:id", "GET", "Get collected media item"},
		{"/media/runs", "GET", "List collection runs"},
		{"/media/runs/:id/cancel", "POST", "Cancel collection run"},
		{"/media/schedules", "GET", "List collection schedules"},
		{"/media/schedules/:id", "PUT", "Update collection schedule"},
		{"/media/sources/:id/health-history", "GET", "List source health history"},
		{"/media/items/:id/refresh", "POST", "Refresh playback addresses"},
		{"/media/episodes/:id/check", "POST", "Probe playback address"},
		{"/media/episodes/:id/stream", "GET", "Stream playback through the CMS"},
	}
	for _, endpoint := range endpoints {
		api := model.SysApi{
			Path:   strings.TrimRight(prefix, "/") + endpoint.suffix,
			Method: endpoint.method, ApiGroup: "media", Description: endpoint.desc,
		}
		if err := tx.Where("path = ? AND method = ?", api.Path, api.Method).FirstOrCreate(&api).Error; err != nil {
			return err
		}
		if adminCount == 0 {
			continue
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.SysAuthorityApi{AuthorityId: 1, ApiId: api.ID}).Error; err != nil {
			return err
		}
		rule := model.SysCasbinRule{Ptype: "p", V0: strconv.Itoa(1), V1: api.Path, V2: endpoint.method}
		if err := tx.Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", rule.Ptype, rule.V0, rule.V1, rule.V2).FirstOrCreate(&rule).Error; err != nil {
			return err
		}
	}
	if err := tx.AutoMigrate(
		&media.Source{}, &media.Item{}, &media.SourceItem{}, &media.PlayLine{},
		&media.PlayEpisode{}, &media.CollectionRun{}, &media.CollectionPageFailure{}, &media.HealthCheck{},
		&media.CollectionSchedule{},
	); err != nil {
		return err
	}
	return seedMediaSources(tx)
}

func seedMediaSources(tx *gorm.DB) error {
	defaults := []media.Source{
		{Name: "暴风资源", APIURL: "https://bfzyapi.com/api.php/provide/vod/", APIType: "json", Enabled: true, Priority: 10, RequestIntervalSeconds: 3, TimeoutSeconds: 15, DenyCategories: "伦理片,伦理,成人,色情,短视频,直播"},
		{Name: "无尽资源", APIURL: "https://api.wujinapi.com/api.php/provide/vod/", APIType: "json", Enabled: true, Priority: 20, RequestIntervalSeconds: 3, TimeoutSeconds: 15, DenyCategories: "伦理片,伦理,成人,色情,短视频,直播"},
		{Name: "豪华资源", APIURL: "https://hhzyapi.com/api.php/provide/vod/at/json", APIType: "json", Enabled: true, Priority: 30, RequestIntervalSeconds: 3, TimeoutSeconds: 15, DenyCategories: "伦理片,伦理,成人,色情,短视频,直播"},
	}
	for _, source := range defaults {
		var existing media.Source
		err := tx.Where("api_url = ?", source.APIURL).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := tx.Create(&source).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
	}
	return nil
}
