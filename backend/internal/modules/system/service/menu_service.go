package service

import (
	"context"
	"errors"

	logger "github.com/CIPFZ/video-cms/internal/core/log"
	"github.com/CIPFZ/video-cms/internal/modules/system/dto"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/CIPFZ/video-cms/internal/modules/system/repository"
	"github.com/CIPFZ/video-cms/internal/svc"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type IMenuService interface {
	GetUserMenuTree(ctx context.Context, authorityId uint) ([]model.SysMenu, error)
	GetMenuList(ctx context.Context) ([]model.SysMenu, error)
	AddBaseMenu(ctx context.Context, req dto.AddMenuReq) error
	UpdateBaseMenu(ctx context.Context, req dto.UpdateMenuReq) error
	DeleteBaseMenu(ctx context.Context, id uint) error
	GetMenuAuthority(ctx context.Context, authorityId uint) ([]model.SysMenu, error)
}

type MenuService struct {
	svcCtx   *svc.ServiceContext
	menuRepo repository.IMenuRepository
}

func NewMenuService(svcCtx *svc.ServiceContext, menuRepo repository.IMenuRepository) IMenuService {
	return &MenuService{
		svcCtx:   svcCtx,
		menuRepo: menuRepo,
	}
}

func (s *MenuService) GetUserMenuTree(ctx context.Context, authorityId uint) ([]model.SysMenu, error) {
	allMenus, err := s.menuRepo.GetByAuthorityId(ctx, authorityId)
	if err != nil {
		s.svcCtx.Logger.Error("get_menu_tree_repo_error", zap.Error(err))
		return nil, errors.New("获取菜单数据失败")
	}
	return s.buildMenuTree(allMenus), nil
}

func (s *MenuService) GetMenuList(ctx context.Context) ([]model.SysMenu, error) {
	allMenus, err := s.menuRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	return s.buildMenuTree(allMenus), nil
}

func (s *MenuService) AddBaseMenu(ctx context.Context, req dto.AddMenuReq) error {
	log := logger.GetLogger(ctx)

	_, err := s.menuRepo.FindByPath(ctx, req.Path)
	if err == nil {
		log.Warn("添加菜单失败，Path已存在", zap.String("path", req.Path))
		return errors.New("路由Path已存在，请更换")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	menu := model.SysMenu{
		ParentId:   req.ParentId,
		Path:       req.Path,
		Name:       req.Name,
		NameEn:     req.NameEn,
		Component:  req.Component,
		Sort:       req.Sort,
		Icon:       req.Icon,
		HideInMenu: req.HideInMenu,
		Access:     req.Access,
		Target:     req.Target,
		Locale:     req.Locale,
	}

	if err := s.menuRepo.Create(ctx, &menu); err != nil {
		log.Error("创建菜单失败", zap.Error(err))
		return err
	}
	return nil
}

func (s *MenuService) UpdateBaseMenu(ctx context.Context, req dto.UpdateMenuReq) error {
	log := logger.GetLogger(ctx)

	menu, err := s.menuRepo.FindById(ctx, req.ID)
	if err != nil {
		return errors.New("菜单不存在")
	}

	if req.Path != menu.Path {
		if _, err := s.menuRepo.FindByPathExcludeId(ctx, req.Path, req.ID); err == nil {
			return errors.New("路由Path已存在")
		}
	}

	updMap := map[string]interface{}{
		"parent_id":    req.ParentId,
		"path":         req.Path,
		"name":         req.Name,
		"component":    req.Component,
		"sort":         req.Sort,
		"icon":         req.Icon,
		"hide_in_menu": req.HideInMenu,
		"access":       req.Access,
		"target":       req.Target,
		"locale":       req.Locale,
	}

	if req.NameEn != nil {
		updMap["name_en"] = *req.NameEn
	}
	if err := s.menuRepo.Update(ctx, menu, updMap); err != nil {
		log.Error("更新菜单失败", zap.Error(err))
		return err
	}
	return nil
}

func (s *MenuService) DeleteBaseMenu(ctx context.Context, id uint) error {
	log := logger.GetLogger(ctx)

	count, err := s.menuRepo.CountByParentId(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("此菜单存在子菜单，不可删除")
	}

	if err := s.menuRepo.DeleteWithAssociations(ctx, id); err != nil {
		log.Error("删除菜单失败", zap.Error(err))
		return err
	}
	return nil
}

func (s *MenuService) GetMenuAuthority(ctx context.Context, authorityId uint) ([]model.SysMenu, error) {
	return s.menuRepo.GetByAuthorityId(ctx, authorityId)
}

func (s *MenuService) buildMenuTree(menus []model.SysMenu) []model.SysMenu {
	nodes := make(map[uint]model.SysMenu, len(menus))
	childrenByParent := make(map[uint][]uint, len(menus))

	for _, menu := range menus {
		menu.Children = nil
		nodes[menu.ID] = menu
		childrenByParent[menu.ParentId] = append(childrenByParent[menu.ParentId], menu.ID)
	}

	var build func(uint) model.SysMenu
	build = func(id uint) model.SysMenu {
		node := nodes[id]
		for _, childID := range childrenByParent[id] {
			node.Children = append(node.Children, build(childID))
		}
		return node
	}

	rootMenus := make([]model.SysMenu, 0, len(childrenByParent[0]))
	for _, rootID := range childrenByParent[0] {
		rootMenus = append(rootMenus, build(rootID))
	}

	return rootMenus
}
