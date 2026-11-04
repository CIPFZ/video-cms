package seed

import (
	"errors"
	tokenCore "github.com/CIPFZ/video-cms/internal/core/token"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
)

// Add the external catalog without granting it to existing tokens. Only roles
// already permitted to list tokens gain access to the matching options endpoint.
func EnsureTokenEndpoints(tx *gorm.DB, prefix string) error {
	prefix = strings.TrimRight(prefix, "/")
	for path, method := range tokenCore.Endpoints(prefix) {
		api := model.SysApi{Path: path, Method: method, ApiGroup: "external", Description: "Verify API token identity"}
		if err := tx.Where("path = ? AND method = ?", path, method).FirstOrCreate(&api).Error; err != nil {
			return err
		}
	}
	options := model.SysApi{Path: prefix + "/sys/api-token/options", Method: "POST", ApiGroup: "system-api-token", Description: "List token-accessible APIs"}
	if err := tx.Where("path = ? AND method = ?", options.Path, options.Method).FirstOrCreate(&options).Error; err != nil {
		return err
	}
	var rules []model.SysCasbinRule
	if err := tx.Where("ptype = ? AND v1 = ? AND v2 = ?", "p", prefix+"/sys/api-token/getApiTokenList", "POST").Find(&rules).Error; err != nil {
		return err
	}
	for _, rule := range rules {
		copy := model.SysCasbinRule{Ptype: "p", V0: rule.V0, V1: options.Path, V2: options.Method}
		if err := tx.Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", copy.Ptype, copy.V0, copy.V1, copy.V2).FirstOrCreate(&copy).Error; err != nil {
			return err
		}
	}
	var old model.SysApi
	if err := tx.Where("path = ? AND method = ?", prefix+"/sys/api-token/getApiTokenList", "POST").First(&old).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	var grants []model.SysAuthorityApi
	if err := tx.Where("api_id = ?", old.ID).Find(&grants).Error; err != nil {
		return err
	}
	for _, grant := range grants {
		grant.ApiId = options.ID
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&grant).Error; err != nil {
			return err
		}
	}
	return nil
}
