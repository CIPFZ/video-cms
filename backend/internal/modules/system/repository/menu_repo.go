package repository

import (
	"context"
	"errors"
	"github.com/CIPFZ/video-cms/internal/core/claims"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"gorm.io/gorm"
)

// IMenuRepository 定义了菜单数据仓库的接口
type IMenuRepository interface {
	// FindById 根据 ID 查询菜单
	FindById(ctx context.Context, id uint) (*model.SysMenu, error)
	// FindByPath 根据路径查询菜单
	FindByPath(ctx context.Context, path string) (*model.SysMenu, error)
	// FindByPathExcludeId 根据路径查询菜单，并排除指定的 ID
	FindByPathExcludeId(ctx context.Context, path string, excludeId uint) (*model.SysMenu, error)
	// GetAll 获取所有菜单
	GetAll(ctx context.Context) ([]model.SysMenu, error)
	// GetByAuthorityId 根据角色 ID 获取该角色拥有的所有菜单
	GetByAuthorityId(ctx context.Context, authorityId uint) ([]model.SysMenu, error)
	// CountByParentId 统计指定父菜单下的子菜单数量
	CountByParentId(ctx context.Context, parentId uint) (int64, error)

	// Create 创建一个新菜单
	Create(ctx context.Context, menu *model.SysMenu) error
	// Update 更新指定菜单的信息
	Update(ctx context.Context, menu *model.SysMenu, cols map[string]interface{}) error
	// DeleteWithAssociations 删除菜单并级联删除其与角色的关联关系
	DeleteWithAssociations(ctx context.Context, id uint) error
}

// MenuRepository 是 IMenuRepository 的 GORM 实现
type MenuRepository struct {
	db *gorm.DB
}

// NewMenuRepository 创建一个新的 MenuRepository 实例
func NewMenuRepository(db *gorm.DB) IMenuRepository {
	return &MenuRepository{db: db}
}

// FindById 根据 ID 从数据库中查找菜单
func (r *MenuRepository) FindById(ctx context.Context, id uint) (*model.SysMenu, error) {
	var menu model.SysMenu
	err := r.db.WithContext(ctx).First(&menu, id).Error
	return &menu, err
}

// FindByPath 根据路径从数据库中查找菜单
func (r *MenuRepository) FindByPath(ctx context.Context, path string) (*model.SysMenu, error) {
	var menu model.SysMenu
	err := r.db.WithContext(ctx).Where("path = ?", path).First(&menu).Error
	return &menu, err
}

// FindByPathExcludeId 根据路径查找菜单，但排除指定的 ID，用于更新时检查路径是否与其它菜单冲突
func (r *MenuRepository) FindByPathExcludeId(ctx context.Context, path string, excludeId uint) (*model.SysMenu, error) {
	var menu model.SysMenu
	err := r.db.WithContext(ctx).Where("path = ? AND id != ?", path, excludeId).First(&menu).Error
	return &menu, err
}

// GetAll 从数据库中获取所有菜单，并按 'sort' 字段排序
func (r *MenuRepository) GetAll(ctx context.Context) ([]model.SysMenu, error) {
	var menus []model.SysMenu
	err := r.db.WithContext(ctx).Order("sort").Order("id").Find(&menus).Error
	return menus, err
}

// GetByAuthorityId 通过联表查询获取特定角色拥有的所有菜单
func (r *MenuRepository) GetByAuthorityId(ctx context.Context, authorityId uint) ([]model.SysMenu, error) {
	var menus []model.SysMenu
	// 通过 sys_authority_menus 关联表进行 JOIN 查询
	err := r.db.WithContext(ctx).
		Table("sys_menus").
		Select("sys_menus.*").
		Joins("JOIN sys_authority_menus ON sys_menus.id = sys_authority_menus.menu_id").
		Where("sys_authority_menus.authority_id = ?", authorityId).
		Where("sys_menus.deleted_at IS NULL"). // 确保菜单未被软删除
		Order("sys_menus.sort").
		Order("sys_menus.id").
		Find(&menus).Error
	return menus, err
}

// CountByParentId 统计指定父菜单下的子菜单数量
func (r *MenuRepository) CountByParentId(ctx context.Context, parentId uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.SysMenu{}).Where("parent_id = ?", parentId).Count(&count).Error
	return count, err
}

// Create 在数据库中创建一个新的菜单记录
func (r *MenuRepository) Create(ctx context.Context, menu *model.SysMenu) error {
	return claims.PolicyTransaction(ctx, r.db, func(tx *gorm.DB) error {
		if err := validateMenu(tx, menu.ID, menu.ParentId, menu.Path); err != nil {
			return err
		}
		return tx.Create(menu).Error
	})
}

// Update 使用给定的列更新数据库中的菜单记录
func (r *MenuRepository) Update(ctx context.Context, menu *model.SysMenu, cols map[string]interface{}) error {
	return claims.PolicyTransaction(ctx, r.db, func(tx *gorm.DB) error {
		if err := tx.First(menu, menu.ID).Error; err != nil {
			return err
		}
		parent := menu.ParentId
		path := menu.Path
		if value, ok := cols["parent_id"].(uint); ok {
			parent = value
		}
		if value, ok := cols["path"].(string); ok {
			path = value
		}
		if err := validateMenu(tx, menu.ID, parent, path); err != nil {
			return err
		}
		return tx.Model(menu).Updates(cols).Error
	})
}

// DeleteWithAssociations 在事务中删除菜单及其关联数据
func (r *MenuRepository) DeleteWithAssociations(ctx context.Context, id uint) error {
	return claims.PolicyTransaction(ctx, r.db, func(tx *gorm.DB) error {
		var children int64
		if err := tx.Model(&model.SysMenu{}).Where("parent_id = ?", id).Count(&children).Error; err != nil {
			return err
		}
		if children > 0 {
			return errors.New("请先删除子菜单")
		}
		// 1. 硬删除 sys_authority_menus 关联表中的相关记录
		if err := tx.Table("sys_authority_menus").
			Where("menu_id = ?", id).
			Delete(nil).Error; err != nil {
			return err
		}
		// 2. 软删除菜单本身
		if err := tx.Delete(&model.SysMenu{}, id).Error; err != nil {
			return err
		}
		return nil
	})
}

func validateMenu(tx *gorm.DB, id, parent uint, path string) error {
	var count int64
	if err := tx.Model(&model.SysMenu{}).Where("id <> ? AND path = ?", id, path).Count(&count).Error; err != nil {
		return err
	}
	if count != 0 {
		return errors.New("菜单路径已存在")
	}
	seen := map[uint]bool{}
	if id != 0 {
		seen[id] = true
	}
	for parent != 0 {
		if seen[parent] {
			return errors.New("菜单不能形成循环")
		}
		seen[parent] = true
		var item model.SysMenu
		if err := tx.First(&item, parent).Error; err != nil {
			return errors.New("父菜单不存在")
		}
		parent = item.ParentId
	}
	return nil
}
