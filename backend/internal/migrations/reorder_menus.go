package migrations

import (
	"time"

	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/CIPFZ/video-cms/internal/modules/system/seed"
	"gorm.io/gorm"
)

func reorderCMSMenus(db *gorm.DB) error {
	const version = "20260910_cms_menu_order"
	if err := db.AutoMigrate(&schemaMigration{}); err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&schemaMigration{}).Where("name = ?", version).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		for path, order := range seed.MenuOrder {
			if err := tx.Model(&model.SysMenu{}).Where("path = ?", path).Update("sort", order).Error; err != nil {
				return err
			}
		}
		return tx.Create(&schemaMigration{Name: version, AppliedAt: time.Now()}).Error
	})
}
