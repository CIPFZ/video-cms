package repository

import (
	"context"
	"errors"
	"github.com/CIPFZ/video-cms/internal/core/claims"
	"strconv"

	"github.com/CIPFZ/video-cms/internal/modules/system/model"

	"gorm.io/gorm"
)

type IAuthorityRepository interface {
	// 查询
	GetAll(ctx context.Context) ([]model.SysAuthority, int64, error)
	FindById(ctx context.Context, authorityId uint) (*model.SysAuthority, error)
	CountByParentId(ctx context.Context, parentId uint) (int64, error)
	CountUserUsage(ctx context.Context, authorityId uint) (int64, error)

	// 写入
	Create(ctx context.Context, auth *model.SysAuthority) error
	Update(ctx context.Context, auth *model.SysAuthority, cols map[string]interface{}) error
	Delete(ctx context.Context, authorityId uint) error

	// 关联操作
	SetMenuAuthority(ctx context.Context, authorityId uint, menuIds []uint) error
}

type AuthorityRepository struct {
	db *gorm.DB
}

func NewAuthorityRepository(db *gorm.DB) IAuthorityRepository {
	return &AuthorityRepository{db: db}
}

// GetAll 获取所有角色 (用于构建树)
func (r *AuthorityRepository) GetAll(ctx context.Context) ([]model.SysAuthority, int64, error) {
	var list []model.SysAuthority
	var total int64

	db := r.db.WithContext(ctx).Model(&model.SysAuthority{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *AuthorityRepository) FindById(ctx context.Context, authorityId uint) (*model.SysAuthority, error) {
	var auth model.SysAuthority
	// 使用 authority_id 字段查询 (因为是自定义ID)
	err := r.db.WithContext(ctx).Where("authority_id = ?", authorityId).First(&auth).Error
	return &auth, err
}

func (r *AuthorityRepository) CountByParentId(ctx context.Context, parentId uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.SysAuthority{}).Where("parent_id = ?", parentId).Count(&count).Error
	return count, err
}

// CountUserUsage 检查角色是否被用户使用
func (r *AuthorityRepository) CountUserUsage(ctx context.Context, authorityId uint) (int64, error) {
	var count int64
	// 查询 sys_user_authorities 表
	err := r.db.WithContext(ctx).Table("sys_user_authorities").Where("authority_id = ?", authorityId).Count(&count).Error
	return count, err
}

func (r *AuthorityRepository) Create(ctx context.Context, auth *model.SysAuthority) error {
	return claims.PolicyTransaction(ctx, r.db, func(tx *gorm.DB) error {
		if auth.ParentId != 0 {
			if auth.ParentId == auth.AuthorityId {
				return errors.New("角色不能是自身的父角色")
			}
			var parent model.SysAuthority
			if err := tx.Where("authority_id = ? AND deleted_at IS NULL", auth.ParentId).First(&parent).Error; err != nil {
				return errors.New("父角色不存在")
			}
		}
		return tx.Create(auth).Error
	})
}

func (r *AuthorityRepository) Update(ctx context.Context, auth *model.SysAuthority, cols map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(auth).Updates(cols).Error
}

func (r *AuthorityRepository) Delete(ctx context.Context, authorityId uint) error {
	if authorityId == 1 {
		return errors.New("管理员角色不可删除")
	}
	return claims.PolicyTransaction(ctx, r.db, func(tx *gorm.DB) error {
		var children int64
		if err := tx.Model(&model.SysAuthority{}).Where("parent_id = ?", authorityId).Count(&children).Error; err != nil {
			return err
		}
		if children > 0 {
			return errors.New("请先删除子角色")
		}
		var used int64
		if err := tx.Model(&model.SysUser{}).Where("authority_id = ?", authorityId).Count(&used).Error; err != nil {
			return err
		}
		if used > 0 {
			return errors.New("角色仍被使用")
		}
		if err := tx.Model(&model.SysUserAuthority{}).Where("authority_id = ?", authorityId).Count(&used).Error; err != nil {
			return err
		}
		if used > 0 {
			return errors.New("角色仍被使用")
		}
		for _, entity := range []any{&model.SysAuthorityMenu{}, &model.SysAuthorityApi{}} {
			if err := tx.Where("authority_id = ?", authorityId).Delete(entity).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("v0 = ?", strconv.FormatUint(uint64(authorityId), 10)).Delete(&model.SysCasbinRule{}).Error; err != nil {
			return err
		}
		return tx.Where("authority_id = ?", authorityId).Delete(&model.SysAuthority{}).Error
	})
}

// SetMenuAuthority 设置角色菜单权限 (事务)
func (r *AuthorityRepository) SetMenuAuthority(ctx context.Context, authorityId uint, menuIds []uint) error {
	return claims.PolicyTransaction(ctx, r.db, func(tx *gorm.DB) error {
		var role model.SysAuthority
		if err := tx.Where("authority_id = ? AND deleted_at IS NULL", authorityId).First(&role).Error; err != nil {
			return err
		}
		unique := make(map[uint]bool)
		for _, id := range menuIds {
			unique[id] = true
		}
		menuIds = menuIds[:0]
		for id := range unique {
			menuIds = append(menuIds, id)
		}
		if len(menuIds) > 0 {
			var count int64
			if err := tx.Model(&model.SysMenu{}).Where("id IN ?", menuIds).Count(&count).Error; err != nil {
				return err
			}
			if count != int64(len(menuIds)) {
				return errors.New("菜单不存在")
			}
		}
		// 1. 硬删除旧关联
		if err := tx.Table("sys_authority_menus").
			Where("authority_id = ?", authorityId).
			Delete(nil).Error; err != nil {
			return err
		}

		// 2. 批量插入新关联
		if len(menuIds) > 0 {
			var relations []model.SysAuthorityMenu
			for _, menuId := range menuIds {
				relations = append(relations, model.SysAuthorityMenu{
					AuthorityId: authorityId,
					MenuId:      menuId,
				})
			}
			if err := tx.Create(&relations).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
