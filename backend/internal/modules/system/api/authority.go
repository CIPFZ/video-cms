package api

import (
	logger "github.com/CIPFZ/video-cms/internal/core/log"
	"github.com/CIPFZ/video-cms/internal/modules/common"
	"github.com/CIPFZ/video-cms/internal/modules/system/dto"
	"github.com/CIPFZ/video-cms/internal/modules/system/service"
	"github.com/CIPFZ/video-cms/internal/svc"
	"github.com/CIPFZ/video-cms/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuthorityApi 提供了角色（权限）管理的相关接口
type AuthorityApi struct {
	svcCtx      *svc.ServiceContext
	authService service.IAuthorityService
}

// NewAuthorityApi 创建一个新的 AuthorityApi 实例
func NewAuthorityApi(svcCtx *svc.ServiceContext, authService service.IAuthorityService) *AuthorityApi {
	return &AuthorityApi{
		svcCtx:      svcCtx,
		authService: authService,
	}
}

// GetAuthorityList 获取角色列表（以树状结构返回）
// @Tags Authority
// @Summary 获取角色列表
// @Security ApiKeyAuth
// @Produce application/json
// @Param data body dto.PageInfo false "分页参数（当前版本返回全量树，可忽略）"
// @Success 200 {object} response.Response{data=dto.PageResult{list=[]model.SysAuthority}} "成功, 返回角色树"
// @Router /authority/getAuthorityList [post]
func (a *AuthorityApi) GetAuthorityList(c *gin.Context) {
	var pageInfo common.PageInfo
	_ = c.ShouldBindJSON(&pageInfo) // 允许请求体为空

	log := logger.GetLogger(c)
	list, total, err := a.authService.GetAuthorityList(c.Request.Context(), pageInfo)
	if err != nil {
		log.Error("get_authority_list_error", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}
	// 返回树状结构，通常不分页，所以 Page 和 PageSize 硬编码以适应 PageResult 结构
	response.OkWithDetailed(common.PageResult{
		List:     list,
		Total:    total,
		Page:     1,
		PageSize: int(total),
	}, "获取成功", c)
}

// CreateAuthority 创建一个新的角色
// @Tags Authority
// @Summary 创建角色
// @Security ApiKeyAuth
// @Produce application/json
// @Param data body dto.CreateAuthorityReq true "角色信息"
// @Success 200 {object} response.Response{} "创建成功"
// @Router /authority/createAuthority [post]
func (a *AuthorityApi) CreateAuthority(c *gin.Context) {
	var req dto.CreateAuthorityReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}
	log := logger.GetLogger(c)
	if err := a.authService.CreateAuthority(c.Request.Context(), req); err != nil {
		log.Error("create_authority_error", zap.Error(err))
		response.FailWithMessage("创建失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("创建成功", c)
}

// UpdateAuthority 更新一个已存在的角色
// @Tags Authority
// @Summary 更新角色
// @Security ApiKeyAuth
// @Produce application/json
// @Param data body dto.UpdateAuthorityReq true "角色信息"
// @Success 200 {object} response.Response{} "更新成功"
// @Router /authority/updateAuthority [post]
func (a *AuthorityApi) UpdateAuthority(c *gin.Context) {
	var req dto.UpdateAuthorityReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}
	log := logger.GetLogger(c)
	if err := a.authService.UpdateAuthority(c.Request.Context(), req); err != nil {
		log.Error("update_authority_error", zap.Error(err))
		response.FailWithMessage("更新失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("更新成功", c)
}

// DeleteAuthority 删除一个角色
// @Tags Authority
// @Summary 删除角色
// @Security ApiKeyAuth
// @Produce application/json
// @Param data body dto.GetByIdReq true "角色ID"
// @Success 200 {object} response.Response{} "删除成功"
// @Router /authority/deleteAuthority [post]
func (a *AuthorityApi) DeleteAuthority(c *gin.Context) {
	var req common.GetByIdReq // 复用通用的按ID获取的请求结构
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}
	log := logger.GetLogger(c)
	// 调用 Service 层执行删除，Service 层会处理业务逻辑检查（如是否存在子角色）
	if err := a.authService.DeleteAuthority(c.Request.Context(), req.Uint()); err != nil {
		log.Error("delete_authority_error", zap.Error(err))
		// 将 Service 层返回的业务错误直接展示给前端
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

// SetAuthorityMenus 设置角色的菜单权限
// @Tags Authority
// @Summary 设置角色的菜单权限
// @Security ApiKeyAuth
// @Produce application/json
// @Param data body dto.SetAuthorityMenusReq true "角色ID和菜单ID列表"
// @Success 200 {object} response.Response{} "设置成功"
// @Router /authority/setDataAuthority [post]
func (a *AuthorityApi) SetAuthorityMenus(c *gin.Context) {
	var req dto.SetAuthorityMenusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}
	log := logger.GetLogger(c)
	if err := a.authService.SetAuthorityMenus(c.Request.Context(), req); err != nil {
		log.Error("set_authority_menu_error", zap.Error(err))
		response.FailWithMessage("设置失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("设置成功", c)
}
