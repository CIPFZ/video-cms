package api

import (
	"github.com/CIPFZ/video-cms/internal/modules/common"
	"github.com/CIPFZ/video-cms/internal/modules/system/dto"
	"github.com/CIPFZ/video-cms/internal/modules/system/service"
	"github.com/CIPFZ/video-cms/internal/svc"
	"github.com/CIPFZ/video-cms/pkg/response"
	"github.com/CIPFZ/video-cms/pkg/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// MenuApi 提供了菜单管理的相关接口
type MenuApi struct {
	svcCtx      *svc.ServiceContext
	menuService service.IMenuService
}

// NewMenuApi 创建一个新的 MenuApi 实例
func NewMenuApi(svcCtx *svc.ServiceContext, menuService service.IMenuService) *MenuApi {
	return &MenuApi{
		svcCtx:      svcCtx,
		menuService: menuService,
	}
}

// GetMenu 获取当前用户的动态菜单树
// @Tags Menu
// @Summary 获取当前用户的动态菜单
// @Security ApiKeyAuth
// @Produce application/json
// @Success 200 {object} response.Response{data=[]dto.SysMenuTree} "成功, 返回菜单树"
// @Router /menu/getMenu [get]
func (a *MenuApi) GetMenu(c *gin.Context) {
	// 1. 从 JWT Claims 中获取当前用户的角色 ID
	authorityId := utils.GetAuthorityId(c)
	if authorityId == 0 {
		response.FailWithMessage("无法获取用户角色信息, 请重新登录", c)
		return
	}

	// 2. 调用 Service 层获取该角色的菜单树
	menus, err := a.menuService.GetUserMenuTree(c.Request.Context(), authorityId)
	if err != nil {
		// 错误日志已在 Service 层记录
		response.FailWithMessage("获取菜单失败: "+err.Error(), c)
		return
	}

	// 3. 返回菜单树给前端
	response.OkWithData(menus, c)
}

// GetMenuList 获取所有菜单的列表（通常用于菜单管理界面）
// @Tags Menu
// @Summary 获取所有菜单列表（树状结构）
// @Security ApiKeyAuth
// @Produce application/json
// @Success 200 {object} response.Response{data=[]model.SysMenu} "成功, 返回所有菜单的树状列表"
// @Router /menu/getMenuList [post]
func (a *MenuApi) GetMenuList(c *gin.Context) {
	// GVA 习惯用 POST 做查询，RESTful 建议 GET，这里兼容前端 API 定义
	menus, err := a.menuService.GetMenuList(c.Request.Context())
	if err != nil {
		a.svcCtx.Logger.Error("get_menu_list_error", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithData(menus, c)
}

// AddBaseMenu 添加一个新的基础菜单
// @Tags Menu
// @Summary 新增基础菜单
// @Security ApiKeyAuth
// @Produce application/json
// @Param data body dto.AddMenuReq true "菜单信息"
// @Success 200 {object} response.Response{} "成功"
// @Router /menu/addBaseMenu [post]
func (a *MenuApi) AddBaseMenu(c *gin.Context) {
	var req dto.AddMenuReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}

	if err := a.menuService.AddBaseMenu(c.Request.Context(), req); err != nil {
		a.svcCtx.Logger.Error("add_menu_error", zap.Error(err))
		response.FailWithMessage("添加失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("添加成功", c)
}

// DeleteBaseMenu 删除一个基础菜单
// @Tags Menu
// @Summary 删除基础菜单
// @Security ApiKeyAuth
// @Produce application/json
// @Param data body dto.GetByIdReq true "菜单ID"
// @Success 200 {object} response.Response{} "成功"
// @Router /menu/deleteBaseMenu [post]
func (a *MenuApi) DeleteBaseMenu(c *gin.Context) {
	var req common.GetByIdReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}

	if err := a.menuService.DeleteBaseMenu(c.Request.Context(), req.Uint()); err != nil {
		a.svcCtx.Logger.Error("delete_menu_error", zap.Error(err))
		// 直接将 Service 层的业务错误（如“存在子菜单”）返回给前端
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

// UpdateBaseMenu 更新一个基础菜单
// @Tags Menu
// @Summary 更新基础菜单
// @Security ApiKeyAuth
// @Produce application/json
// @Param data body dto.UpdateMenuReq true "菜单信息"
// @Success 200 {object} response.Response{} "成功"
// @Router /menu/updateBaseMenu [post]
func (a *MenuApi) UpdateBaseMenu(c *gin.Context) {
	var req dto.UpdateMenuReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}

	if err := a.menuService.UpdateBaseMenu(c.Request.Context(), req); err != nil {
		a.svcCtx.Logger.Error("update_menu_error", zap.Error(err))
		response.FailWithMessage("更新失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("更新成功", c)
}

// GetMenuAuthority 获取指定角色的菜单权限（通常返回所有菜单，并标记该角色已拥有的菜单）
// @Tags Menu
// @Summary 获取角色的菜单权限
// @Security ApiKeyAuth
// @Produce application/json
// @Param data body dto.GetAuthorityIdReq true "角色ID"
// @Success 200 {object} response.Response{data=dto.MenuAuthorityResp} "成功"
// @Router /menu/getMenuAuthority [post]
func (a *MenuApi) GetMenuAuthority(c *gin.Context) {
	var req dto.GetAuthorityIdReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}

	menus, err := a.menuService.GetMenuAuthority(c.Request.Context(), req.AuthorityId)
	if err != nil {
		a.svcCtx.Logger.Error("get_menu_authority_error", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}

	response.OkWithData(menus, c)
}
