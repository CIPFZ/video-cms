package migrations

import (
	"fmt"
	"strings"
	"time"

	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type schemaMigration struct {
	Name      string `gorm:"primaryKey;size:128"`
	AppliedAt time.Time
}

func (schemaMigration) TableName() string { return "sys_schema_migrations" }

// These tables belonged exclusively to the removed demo modules, including
// sys_products and sys_departments. Drop children before their parents.
var legacyModuleTables = []string{
	"plugin_compatible_products", "plugin_release_events", "plugin_releases",
	"plugins", "sys_products", "sys_departments",
	"poem_tag_rel", "poem_work", "poem_author", "meta_tag", "meta_genre", "meta_dynasty",
}

func legacyPage(path string) bool {
	path = strings.TrimPrefix(path, "/")
	for _, root := range []string{"plugin", "plugins", "poetry", "sys/plugin-master"} {
		if path == root || strings.HasPrefix(path, root+"/") {
			return true
		}
	}
	return false
}

func cleanupLegacyModules(db *gorm.DB, routerPrefix string) error {
	const version = "20260910_minimal_cms"
	if err := db.AutoMigrate(&schemaMigration{}); err != nil {
		return err
	}
	var count int64
	if err := db.Model(&schemaMigration{}).Where("name = ?", version).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return cleanupLegacyMetadata(tx, routerPrefix)
	}); err != nil {
		return err
	}
	// MySQL DDL commits implicitly. Metadata cleanup and table drops are retry
	// safe; record completion only after every table has been dropped.
	for _, table := range legacyModuleTables {
		if tx := db.Migrator(); tx.HasTable(table) {
			if err := tx.DropTable(table); err != nil {
				return fmt.Errorf("drop %s: %w", table, err)
			}
		}
	}
	return db.Create(&schemaMigration{Name: version, AppliedAt: time.Now()}).Error
}

func cleanupLegacyMetadata(tx *gorm.DB, routerPrefix string) error {
	roles := []uint{10010, 10013}
	var menus []model.SysMenu
	if err := tx.Unscoped().Find(&menus).Error; err != nil {
		return err
	}
	removed := map[uint]bool{}
	// Include descendants even if an administrator renamed their paths.
	for changed := true; changed; {
		changed = false
		for _, menu := range menus {
			if !removed[menu.ID] && (legacyPage(menu.Path) || legacyPage(menu.Component) || removed[menu.ParentId]) {
				removed[menu.ID], changed = true, true
			}
		}
	}
	var menuIDs []uint
	for id := range removed {
		menuIDs = append(menuIDs, id)
	}
	if len(menuIDs) > 0 {
		if err := tx.Where("menu_id IN ?", menuIDs).Delete(&model.SysAuthorityMenu{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Delete(&model.SysMenu{}, menuIDs).Error; err != nil {
			return err
		}
	}

	var apis []model.SysApi
	if err := tx.Unscoped().Find(&apis).Error; err != nil {
		return err
	}
	isLegacyAPI := func(path string) bool {
		for _, prefix := range []string{strings.TrimRight(routerPrefix, "/"), "/api/v1", ""} {
			if strings.HasPrefix(path, prefix+"/") && legacyPage(strings.TrimPrefix(path, prefix)) {
				return true
			}
		}
		return false
	}
	var apiIDs []uint
	for _, api := range apis {
		if isLegacyAPI(api.Path) {
			apiIDs = append(apiIDs, api.ID)
		}
	}
	if len(apiIDs) > 0 {
		for _, entity := range []any{&model.SysAuthorityApi{}, &model.SysApiTokenApi{}} {
			if err := tx.Where("api_id IN ?", apiIDs).Delete(entity).Error; err != nil {
				return err
			}
		}
		if err := tx.Unscoped().Delete(&model.SysApi{}, apiIDs).Error; err != nil {
			return err
		}
	}
	var policies []model.SysCasbinRule
	if err := tx.Find(&policies).Error; err != nil {
		return err
	}
	for _, policy := range policies {
		if isLegacyAPI(policy.V1) || policy.V0 == "10010" || policy.V0 == "10013" ||
			(policy.Ptype == "g" && (policy.V1 == "10010" || policy.V1 == "10013")) {
			if err := tx.Delete(&policy).Error; err != nil {
				return err
			}
		}
	}
	var logs []model.SysOperationLog
	if err := tx.Unscoped().Select("id", "path").Find(&logs).Error; err != nil {
		return err
	}
	for _, entry := range logs {
		if isLegacyAPI(entry.Path) {
			if err := tx.Unscoped().Delete(&entry).Error; err != nil {
				return err
			}
		}
	}

	// Preserve accounts; users whose current role is removed receive the
	// ordinary CMS role, never administrator privileges.
	var users []model.SysUser
	if err := tx.Unscoped().Where("authority_id IN ?", roles).Find(&users).Error; err != nil {
		return err
	}
	if len(users) > 0 {
		fallback := model.SysAuthority{AuthorityId: 888, AuthorityName: "CommonUser", DefaultRouter: "dashboard/workplace"}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&fallback).Error; err != nil {
			return err
		}
		for _, user := range users {
			if err := tx.Unscoped().Model(&user).Update("authority_id", 888).Error; err != nil {
				return err
			}
			link := model.SysUserAuthority{UserId: user.ID, AuthorityId: 888}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&link).Error; err != nil {
				return err
			}
		}
	}
	for _, entity := range []any{&model.SysUserAuthority{}, &model.SysAuthorityMenu{}, &model.SysAuthorityApi{}} {
		if err := tx.Where("authority_id IN ?", roles).Delete(entity).Error; err != nil {
			return err
		}
	}
	if tx.Migrator().HasTable("sys_data_authority_id") {
		if err := tx.Exec("DELETE FROM sys_data_authority_id WHERE sys_authority_authority_id IN ? OR data_authority_id_authority_id IN ?", roles, roles).Error; err != nil {
			return err
		}
	}
	if err := tx.Model(&model.SysAuthority{}).Where("parent_id IN ?", roles).Update("parent_id", 0).Error; err != nil {
		return err
	}
	if err := tx.Unscoped().Where("authority_id IN ?", roles).Delete(&model.SysAuthority{}).Error; err != nil {
		return err
	}
	var authorities []model.SysAuthority
	if err := tx.Find(&authorities).Error; err != nil {
		return err
	}
	for _, authority := range authorities {
		if legacyPage(authority.DefaultRouter) {
			if err := tx.Model(&authority).Update("default_router", "dashboard/workplace").Error; err != nil {
				return err
			}
		}
	}
	return nil
}
