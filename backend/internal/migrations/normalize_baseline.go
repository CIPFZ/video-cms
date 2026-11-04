package migrations

import (
	"regexp"
	"time"

	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/CIPFZ/video-cms/internal/modules/system/seed"
	"gorm.io/gorm"
)

// Clean up accounts created by the old smoke/browser tests once. Match their
// exact naming convention so upgrading the reusable framework preserves real
// team accounts and accounts created after this migration has completed.
func normalizeCMSBaseline(db *gorm.DB) error {
	const version = "20260910_cms_names_and_test_accounts"
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
		for key, name := range seed.MenuNames {
			if err := tx.Unscoped().Model(&model.SysMenu{}).Where("name = ?", key).Update("name", name).Error; err != nil {
				return err
			}
		}
		var users []model.SysUser
		if err := tx.Unscoped().Select("id", "username").Find(&users).Error; err != nil {
			return err
		}
		pattern := regexp.MustCompile(`^(e2e_user_(b_)?|smoke_user_)[0-9]+$`)
		var ids []uint
		for _, user := range users {
			if pattern.MatchString(user.Username) {
				ids = append(ids, user.ID)
			}
		}
		if len(ids) > 0 {
			for _, entity := range []any{&model.SysUserAuthority{}, &model.SysNoticeReceiver{}, &model.SysOperationLog{}} {
				if err := tx.Unscoped().Where("user_id IN ?", ids).Delete(entity).Error; err != nil {
					return err
				}
			}
			var tokens []model.SysApiToken
			if err := tx.Unscoped().Where("created_by IN ?", ids).Find(&tokens).Error; err != nil {
				return err
			}
			for _, token := range tokens {
				if err := tx.Where("api_token_id = ?", token.ID).Delete(&model.SysApiTokenApi{}).Error; err != nil {
					return err
				}
				if err := tx.Unscoped().Delete(&token).Error; err != nil {
					return err
				}
			}
			// Keep real notices readable even if a test account created one.
			if err := tx.Model(&model.SysNotice{}).Where("created_by IN ?", ids).Update("created_by", 0).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().Delete(&model.SysUser{}, ids).Error; err != nil {
				return err
			}
		}
		var notices []model.SysNotice
		if err := tx.Unscoped().Select("id", "title").Find(&notices).Error; err != nil {
			return err
		}
		testNotice := regexp.MustCompile(`^(E2E (Admin )?Notice |Smoke notice )[0-9]+$`)
		for _, notice := range notices {
			if testNotice.MatchString(notice.Title) {
				if err := tx.Unscoped().Where("notice_id = ?", notice.ID).Delete(&model.SysNoticeReceiver{}).Error; err != nil {
					return err
				}
				if err := tx.Unscoped().Delete(&notice).Error; err != nil {
					return err
				}
			}
		}
		return tx.Create(&schemaMigration{Name: version, AppliedAt: time.Now()}).Error
	})
}
