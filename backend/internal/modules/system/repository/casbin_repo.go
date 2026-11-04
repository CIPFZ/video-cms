package repository

import (
	"context"
	"errors"
	"github.com/CIPFZ/video-cms/internal/core/claims"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"gorm.io/gorm"
	"strconv"
	"strings"
)

type ICasbinRepository interface {
	ReplacePolicy(context.Context, string, [][]string) error
	GetPolicy(context.Context, string) ([][]string, error)
}
type CasbinRepository struct{ db *gorm.DB }

func NewCasbinRepository(db *gorm.DB) ICasbinRepository { return &CasbinRepository{db: db} }

func (r *CasbinRepository) ReplacePolicy(ctx context.Context, subject string, rules [][]string) error {
	roleID, err := strconv.ParseUint(subject, 10, 64)
	if err != nil || roleID == 0 {
		return errors.New("角色无效")
	}
	return claims.PolicyTransaction(ctx, r.db, func(tx *gorm.DB) error {
		var role model.SysAuthority
		if err := tx.Where("authority_id = ? AND deleted_at IS NULL", roleID).First(&role).Error; err != nil {
			return err
		}
		policies := make([]model.SysCasbinRule, 0, len(rules))
		relations := make([]model.SysAuthorityApi, 0, len(rules))
		seen := map[string]bool{}
		for _, rule := range rules {
			if len(rule) != 3 || rule[0] != subject {
				return errors.New("策略格式无效")
			}
			path, method := strings.TrimSpace(rule[1]), strings.ToUpper(strings.TrimSpace(rule[2]))
			key := method + " " + path
			if seen[key] {
				continue
			}
			seen[key] = true
			var api model.SysApi
			if err := tx.Where("path = ? AND method = ?", path, method).First(&api).Error; err != nil {
				return errors.New("策略引用了不存在的 API")
			}
			policies = append(policies, model.SysCasbinRule{Ptype: "p", V0: subject, V1: path, V2: method})
			relations = append(relations, model.SysAuthorityApi{AuthorityId: uint(roleID), ApiId: api.ID})
		}
		if err := tx.Where("v0 = ? AND ptype = ?", subject, "p").Delete(&model.SysCasbinRule{}).Error; err != nil {
			return err
		}
		if err := tx.Where("authority_id = ?", roleID).Delete(&model.SysAuthorityApi{}).Error; err != nil {
			return err
		}
		if len(policies) > 0 {
			if err := tx.Create(&policies).Error; err != nil {
				return err
			}
			if err := tx.Create(&relations).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
func (r *CasbinRepository) GetPolicy(ctx context.Context, subject string) ([][]string, error) {
	var rules []model.SysCasbinRule
	if err := r.db.WithContext(ctx).Where("v0 = ? AND ptype = ?", subject, "p").Order("id").Find(&rules).Error; err != nil {
		return nil, err
	}
	result := make([][]string, 0, len(rules))
	for _, rule := range rules {
		result = append(result, []string{rule.V0, rule.V1, rule.V2})
	}
	return result, nil
}
