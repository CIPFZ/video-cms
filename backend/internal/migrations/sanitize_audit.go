package migrations

import (
	"time"

	"github.com/CIPFZ/video-cms/internal/core/audit"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"gorm.io/gorm"
)

func sanitizeAuditHistory(db *gorm.DB) error {
	const version = "20260910_redact_audit_bodies"
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
		var records []model.SysOperationLog
		err := tx.Unscoped().Select("id", "body", "resp").FindInBatches(&records, 100, func(_ *gorm.DB, _ int) error {
			for _, record := range records {
				if err := tx.Unscoped().Model(&model.SysOperationLog{}).Where("id = ?", record.ID).Updates(map[string]any{
					"body": audit.SanitizeBody([]byte(record.Body)),
					"resp": audit.SanitizeBody([]byte(record.Resp)),
				}).Error; err != nil {
					return err
				}
			}
			return nil
		}).Error
		if err != nil {
			return err
		}
		return tx.Create(&schemaMigration{Name: version, AppliedAt: time.Now()}).Error
	})
}
