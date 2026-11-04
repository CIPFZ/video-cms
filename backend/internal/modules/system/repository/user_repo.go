package repository

import (
	"context"
	"errors"
	"github.com/CIPFZ/video-cms/internal/core/claims"
	"github.com/CIPFZ/video-cms/internal/modules/system/dto"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// IUserRepository 定义用户数据访问接口
type IUserRepository interface {
	FindById(ctx context.Context, id uint) (*model.SysUser, error)
	FindByUuid(ctx context.Context, uuid uuid.UUID) (*model.SysUser, error)
	FindByUsername(ctx context.Context, username string) (*model.SysUser, error)

	GetList(ctx context.Context, req dto.SearchUserReq) ([]model.SysUser, int64, error)

	Create(ctx context.Context, user *model.SysUser) error
	Update(ctx context.Context, user *model.SysUser, columns map[string]interface{}) error
	UpdateColumn(ctx context.Context, uid uuid.UUID, values map[string]interface{}) error

	UpdateWithRoles(ctx context.Context, user *model.SysUser, req dto.UpdateUserReq) error
	DeleteWithAssociations(ctx context.Context, id uint) error
	ResetPassword(ctx context.Context, id uint, password string) error
}

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) IUserRepository {
	return &UserRepository{db: db}
}

// FindById 根据ID查询
func (r *UserRepository) FindById(ctx context.Context, id uint) (*model.SysUser, error) {
	var user model.SysUser
	err := r.db.WithContext(ctx).
		Preload("Authority").
		Preload("Authorities").
		First(&user, id).Error
	return &user, err
}

// FindByUuid 根据UUID查询
func (r *UserRepository) FindByUuid(ctx context.Context, uuid uuid.UUID) (*model.SysUser, error) {
	var user model.SysUser
	err := r.db.WithContext(ctx).
		Where("uuid = ?", uuid).
		Preload("Authority").
		Preload("Authorities").
		First(&user).Error
	return &user, err
}

// FindByUsername 根据用户名查询
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*model.SysUser, error) {
	var user model.SysUser
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	return &user, err
}

// GetList 分页查询
func (r *UserRepository) GetList(ctx context.Context, req dto.SearchUserReq) ([]model.SysUser, int64, error) {
	var list []model.SysUser
	var total int64
	db := r.db.WithContext(ctx).Model(&model.SysUser{})

	if req.Username != "" {
		db = db.Where("username LIKE ?", "%"+req.Username+"%")
	}
	if req.NickName != "" {
		db = db.Where("nick_name LIKE ?", "%"+req.NickName+"%")
	}
	if req.Phone != "" {
		db = db.Where("phone LIKE ?", "%"+req.Phone+"%")
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Scopes(req.Paginate()).
		Preload("Authority").
		Preload("Authorities").
		Order("id desc").
		Find(&list).Error

	return list, total, err
}

// Create 创建用户
func (r *UserRepository) Create(ctx context.Context, user *model.SysUser) error {
	return claims.PolicyTransaction(ctx, r.db, func(tx *gorm.DB) error {
		if user.Status != 0 && user.Status != 1 {
			return errors.New("无效的用户状态")
		}
		if len(user.Authorities) == 0 {
			return errors.New("请至少选择一个角色")
		}
		for _, role := range user.Authorities {
			var count int64
			if err := tx.Model(&model.SysAuthority{}).Where("authority_id = ? AND deleted_at IS NULL", role.AuthorityId).Count(&count).Error; err != nil {
				return err
			}
			if count != 1 {
				return errors.New("角色不存在")
			}
		}
		status := user.Status
		if err := tx.Omit("Authorities.*").Create(user).Error; err != nil {
			return err
		}
		if status == 0 {
			return tx.Model(user).Update("status", 0).Error
		}
		return nil
	})
}

// Update 更新指定字段
func (r *UserRepository) Update(ctx context.Context, user *model.SysUser, columns map[string]interface{}) error {
	return claims.PolicyTransaction(ctx, r.db, func(tx *gorm.DB) error {
		if role, ok := columns["authority_id"].(uint); ok {
			var count int64
			if err := tx.Model(&model.SysUserAuthority{}).Where("user_id = ? AND authority_id = ?", user.ID, role).Count(&count).Error; err != nil {
				return err
			}
			if count != 1 {
				return errors.New("您未拥有该角色权限")
			}
		}
		return tx.Model(user).Updates(columns).Error
	})
}

// UpdateWithRoles 事务更新用户及其角色关联
func (r *UserRepository) UpdateWithRoles(ctx context.Context, user *model.SysUser, req dto.UpdateUserReq) error {
	return claims.PolicyTransaction(ctx, r.db, func(tx *gorm.DB) error {

		if req.Status != 0 && req.Status != 1 {
			return errors.New("无效的用户状态")
		}
		if err := tx.First(user, user.ID).Error; err != nil {
			return err
		}
		if user.Username == "admin" {
			hasAdmin := false
			for _, id := range req.AuthorityIds {
				hasAdmin = hasAdmin || id == 1
			}
			if req.Status != 1 || !hasAdmin {
				return errors.New("基础管理员必须保持启用及管理员角色")
			}
		}
		if len(req.AuthorityIds) == 0 {
			return errors.New("请至少选择一个角色")
		}
		for _, id := range req.AuthorityIds {
			var count int64
			if err := tx.Model(&model.SysAuthority{}).Where("authority_id = ? AND deleted_at IS NULL", id).Count(&count).Error; err != nil {
				return err
			}
			if count != 1 {
				return errors.New("角色不存在")
			}
		}
		// 1. 检查 "当前角色" 是否还在新分配的列表中
		// 如果不在，强制重置为新列表的第一个
		isOldAuthValid := false
		authorityId := user.AuthorityID
		for _, id := range req.AuthorityIds {
			if user.AuthorityID == id {
				isOldAuthValid = true
				break
			}
		}
		if !isOldAuthValid {
			// 原来的当前角色被删掉了，重置为新的第一个
			authorityId = req.AuthorityIds[0]
		}
		// 2. 更新基础信息
		updMap := map[string]interface{}{
			"nick_name":     req.NickName,
			"authority_id":  authorityId,
			"phone":         req.Phone,
			"email":         req.Email,
			"status":        req.Status,
			"token_version": gorm.Expr("token_version + 1"),
		}

		// 2. 更新主表
		if err := tx.Model(&user).Updates(updMap).Error; err != nil {
			return err
		}
		// 3. 更新角色关联
		var auths []model.SysAuthority
		for _, id := range req.AuthorityIds {
			auths = append(auths, model.SysAuthority{AuthorityId: id})
		}
		if err := tx.Model(&user).Omit("Authorities.*").Association("Authorities").Replace(auths); err != nil {
			return err
		}

		return nil
	})
}

// DeleteWithAssociations 级联删除
func (r *UserRepository) DeleteWithAssociations(ctx context.Context, id uint) error {
	return claims.PolicyTransaction(ctx, r.db, func(tx *gorm.DB) error {
		var user model.SysUser
		if err := tx.First(&user, id).Error; err != nil {
			return err
		}
		if user.Username == "admin" {
			return errors.New("不能删除基础管理员")
		}
		// 1. 硬删除关联表
		if err := tx.Table("sys_user_authorities").Where("user_id = ?", id).Delete(nil).Error; err != nil {
			return err
		}
		// 2. 软删除用户
		if err := tx.Delete(&model.SysUser{}, id).Error; err != nil {
			return err
		}
		return nil
	})
}

// ResetPassword 重置密码
func (r *UserRepository) ResetPassword(ctx context.Context, id uint, password string) error {
	return r.db.WithContext(ctx).Model(&model.SysUser{}).Where("id = ?", id).Updates(map[string]any{"password": password, "token_version": gorm.Expr("token_version + 1")}).Error
}

// UpdateColumn 实现
func (r *UserRepository) UpdateColumn(ctx context.Context, uid uuid.UUID, values map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&model.SysUser{}).
		Where("uuid = ?", uid).
		Updates(values).Error
}
