package migrations

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/CIPFZ/video-cms/internal/modules/media"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type mediaEndpoint struct {
	suffix string
	method string
	desc   string
}

var mediaRunItemsEndpoint = mediaEndpoint{"/media/runs/:id/items", "GET", "List collection run items"}

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
	endpoints := []mediaEndpoint{
		{"/media/sources", "GET", "List media sources"},
		{"/media/sources", "POST", "Create media source"},
		{"/media/sources/:id", "PUT", "Update media source"},
		{"/media/sources/:id/health", "POST", "Check media source health"},
		{"/media/sources/:id/collect", "POST", "Collect media source records"},
		{"/media/items", "GET", "List collected media items"},
		{"/media/items/:id", "GET", "Get collected media item"},
		{"/media/runs", "GET", "List collection runs"},
		mediaRunItemsEndpoint,
		{"/media/runs/:id/cancel", "POST", "Cancel collection run"},
		{"/media/schedules", "GET", "List collection schedules"},
		{"/media/schedules/:id", "PUT", "Update collection schedule"},
		{"/media/sources/:id/health-history", "GET", "List source health history"},
		{"/media/items/:id/refresh", "POST", "Refresh playback addresses"},
		{"/media/episodes/:id/check", "POST", "Probe playback address"},
		{"/media/episodes/:id/stream", "GET", "Stream playback through the CMS"},
	}
	if err := ensureMediaAPIs(tx, prefix, adminCount > 0, endpoints); err != nil {
		return err
	}
	if err := tx.AutoMigrate(
		&media.Source{}, &media.Item{}, &media.SourceItem{}, &media.PlayLine{},
		&media.PlayEpisode{}, &media.CollectionRun{}, &media.CollectionPageFailure{}, &media.HealthCheck{},
		&media.CollectionSchedule{}, &media.CollectionRunItem{},
	); err != nil {
		return err
	}
	return seedMediaSources(tx)
}

func ensureMediaAPIs(tx *gorm.DB, prefix string, grantAdmin bool, endpoints []mediaEndpoint) error {
	for _, endpoint := range endpoints {
		api := model.SysApi{
			Path:   strings.TrimRight(prefix, "/") + endpoint.suffix,
			Method: endpoint.method, ApiGroup: "media", Description: endpoint.desc,
		}
		if err := tx.Where("path = ? AND method = ?", api.Path, api.Method).FirstOrCreate(&api).Error; err != nil {
			return err
		}
		if !grantAdmin {
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
	return nil
}

// applyMediaRunItems upgrades an installed edition without re-seeding its menus
// or grants: it adds the run item log and its endpoint, and moves untouched
// weekly full-collection defaults to the monthly default.
func applyMediaRunItems(tx *gorm.DB, prefix string) error {
	if err := tx.AutoMigrate(&media.CollectionRunItem{}); err != nil {
		return err
	}
	var adminCount int64
	if err := tx.Model(&model.SysAuthority{}).Where("authority_id = ?", 1).Count(&adminCount).Error; err != nil {
		return err
	}
	if err := ensureMediaAPIs(tx, prefix, adminCount > 0, []mediaEndpoint{mediaRunItemsEndpoint}); err != nil {
		return err
	}
	schedule, err := cron.ParseStandard("CRON_TZ=Asia/Shanghai " + media.DefaultFullCron)
	if err != nil {
		return err
	}
	return tx.Model(&media.CollectionSchedule{}).
		Where("kind = ? AND cron = ? AND enabled = ? AND last_run_at IS NULL", "full", "30 3 * * 0", false).
		Updates(map[string]any{"cron": media.DefaultFullCron, "next_run_at": schedule.Next(time.Now())}).Error
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

const mediaItemsUpdatedIndex = "idx_media_items_updated_at"

// applyMediaPublic indexes the public catalogue's default "recently updated" order.
func applyMediaPublic(tx *gorm.DB, _ string) error {
	if tx.Migrator().HasIndex(&media.Item{}, mediaItemsUpdatedIndex) {
		return nil
	}
	return tx.Exec("CREATE INDEX " + mediaItemsUpdatedIndex + " ON media_items (updated_at)").Error
}
