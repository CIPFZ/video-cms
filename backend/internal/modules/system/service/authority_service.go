package service

import (
	"context"
	"errors"

	"github.com/CIPFZ/video-cms/internal/modules/common"
	"github.com/CIPFZ/video-cms/internal/modules/system/dto"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/CIPFZ/video-cms/internal/modules/system/repository"

	"gorm.io/gorm"
)

// IAuthorityService 定义了角色（权限）管理的服务层接口
type IAuthorityService interface {
	// GetAuthorityList 获取角色列表，以树状结构返回
	GetAuthorityList(ctx context.Context, pageInfo common.PageInfo) ([]model.SysAuthority, int64, error)
	// CreateAuthority 创建一个新的角色
	CreateAuthority(ctx context.Context, req dto.CreateAuthorityReq) error
	// UpdateAuthority 更新一个已存在的角色
	UpdateAuthority(ctx context.Context, req dto.UpdateAuthorityReq) error
	// DeleteAuthority 删除一个角色
	DeleteAuthority(ctx context.Context, authId uint) error
	// SetAuthorityMenus 设置角色的菜单权限
	SetAuthorityMenus(ctx context.Context, req dto.SetAuthorityMenusReq) error
}

// AuthorityService 是 IAuthorityService 的实现
type AuthorityService struct {
	authRepo repository.IAuthorityRepository // 依赖注入 AuthorityRepository
}

// NewAuthorityService 创建一个新的 AuthorityService 实例
func NewAuthorityService(authRepo repository.IAuthorityRepository) IAuthorityService {
	return &AuthorityService{
		authRepo: authRepo,
	}
}

// GetAuthorityList 获取所有角色并构建成树状结构
func (s *AuthorityService) GetAuthorityList(ctx context.Context, pageInfo common.PageInfo) ([]model.SysAuthority, int64, error) {
	// 1. 从仓库获取所有角色的扁平列表
	list, total, err := s.authRepo.GetAll(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 2. 在内存中将扁平列表构建成树状结构
	return s.buildAuthorityTree(list), total, nil
}

// CreateAuthority 创建一个新的角色
func (s *AuthorityService) CreateAuthority(ctx context.Context, req dto.CreateAuthorityReq) error {
	// 1. 检查角色ID是否已存在，防止重复
	_, err := s.authRepo.FindById(ctx, req.AuthorityId)
	if err == nil {
		return errors.New("角色ID已存在")
	}
	// 如果错误不是 "记录未找到"，则说明发生了其他数据库错误
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// 2. 创建新的角色实体并保存
	auth := model.SysAuthority{
		AuthorityId:   req.AuthorityId,
		AuthorityName: req.AuthorityName,
		ParentId:      req.ParentId,
		DefaultRouter: req.DefaultRouter,
	}
	return s.authRepo.Create(ctx, &auth)
}

// UpdateAuthority 更新一个已存在的角色信息
func (s *AuthorityService) UpdateAuthority(ctx context.Context, req dto.UpdateAuthorityReq) error {
	// 1. 检查角色是否存在
	_, err := s.authRepo.FindById(ctx, req.AuthorityId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("要更新的角色不存在")
		}
		return err
	}

	// 2. 构造要更新的字段映射
	updates := map[string]interface{}{
		"authority_name": req.AuthorityName,
		"default_router": req.DefaultRouter,
	}
	// 构造一个只包含 ID 的对象用于 GORM 的 Where 条件
	target := model.SysAuthority{AuthorityId: req.AuthorityId}

	// 3. 调用仓库进行更新
	return s.authRepo.Update(ctx, &target, updates)
}

// DeleteAuthority 删除一个角色，会进行前置检查
func (s *AuthorityService) DeleteAuthority(ctx context.Context, authId uint) error {
	// 1. 检查该角色下是否存在子角色
	childCount, err := s.authRepo.CountByParentId(ctx, authId)
	if err != nil {
		return err
	}
	if childCount > 0 {
		return errors.New("此角色存在子角色，不可删除")
	}

	// 2. 检查该角色是否已被分配给任何用户
	userCount, err := s.authRepo.CountUserUsage(ctx, authId)
	if err != nil {
		return err
	}
	if userCount > 0 {
		return errors.New("此角色有用户正在使用，不可删除")
	}

	// 3. 执行删除操作
	return s.authRepo.Delete(ctx, authId)
}

// SetAuthorityMenus 设置指定角色的菜单权限
func (s *AuthorityService) SetAuthorityMenus(ctx context.Context, req dto.SetAuthorityMenusReq) error {
	// 直接调用仓库层的方法，该方法已包含事务处理
	return s.authRepo.SetMenuAuthority(ctx, req.AuthorityId, req.MenuIds)
}

// buildAuthorityTree 是一个私有辅助函数，用于将角色的扁平列表转换为树状结构
func (s *AuthorityService) buildAuthorityTree(list []model.SysAuthority) []model.SysAuthority {
	// 创建一个映射，键是父ID，值是该父ID下的所有子角色列表
	treeMap := make(map[uint][]*model.SysAuthority)
	for i := range list {
		// 使用指针可以避免在后续操作中复制整个结构体
		treeMap[list[i].ParentId] = append(treeMap[list[i].ParentId], &list[i])
	}

	// 遍历原始列表，为每个角色找到其子角色
	for i := range list {
		// 检查当前角色ID是否在映射中作为父ID存在
		if children, ok := treeMap[list[i].AuthorityId]; ok {
			// 如果存在，将子角色列表（解引用后）赋给当前角色的 Children 字段
			for _, child := range children {
				list[i].Children = append(list[i].Children, *child)
			}
		}
	}

	// 筛选出所有根节点（其 ParentId 为 0）
	var roots []model.SysAuthority
	if rootNodes, ok := treeMap[0]; ok {
		for _, root := range rootNodes {
			roots = append(roots, *root)
		}
	}
	return roots
}
