package api

import (
	logger "github.com/CIPFZ/video-cms/internal/core/log"
	"github.com/CIPFZ/video-cms/internal/modules/system/dto"
	"github.com/CIPFZ/video-cms/internal/modules/system/service"
	"github.com/CIPFZ/video-cms/internal/svc"
	"github.com/CIPFZ/video-cms/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CasbinApi 提供了 Casbin 权限策略管理的相关接口
type CasbinApi struct {
	svcCtx        *svc.ServiceContext
	casbinService service.ICasbinService
}

// NewCasbinApi 创建一个新的 CasbinApi 实例
func NewCasbinApi(svcCtx *svc.ServiceContext, casbinService service.ICasbinService) *CasbinApi {
	return &CasbinApi{
		svcCtx:        svcCtx,
		casbinService: casbinService,
	}
}

// UpdateCasbin 更新角色的 API 访问权限
// @Tags Casbin
// @Summary 更新角色的API权限
// @Security ApiKeyAuth
// @Produce application/json
// @Param data body dto.UpdateCasbinReq true "角色ID和API权限列表"
// @Success 200 {object} response.Response{} "更新成功"
// @Router /casbin/updateCasbin [post]
func (a *CasbinApi) UpdateCasbin(c *gin.Context) {
	var req dto.UpdateCasbinReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}

	// 简单校验
	if req.AuthorityId == "" {
		response.FailWithMessage("角色ID不能为空", c)
		return
	}
	log := logger.GetLogger(c)
	// 调用 Service 层更新 Casbin 策略
	err := a.casbinService.UpdateCasbin(c.Request.Context(), req.AuthorityId, req.CasbinInfos)
	if err != nil {
		log.Error("update_casbin_error", zap.Error(err))
		response.FailWithMessage("更新权限失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("更新成功", c)
}

// GetPolicyPathByAuthorityId 获取指定角色的 API 权限列表
// @Tags Casbin
// @Summary 获取角色的API权限列表
// @Security ApiKeyAuth
// @Produce application/json
// @Param data body dto.GetPolicyPathByAuthorityIdReq true "角色ID"
// @Success 200 {object} response.Response{data=[]dto.CasbinInfo} "成功, 返回权限列表"
// @Router /casbin/getPolicyPathByAuthorityId [post]
func (a *CasbinApi) GetPolicyPathByAuthorityId(c *gin.Context) {
	var req dto.GetPolicyPathByAuthorityIdReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}
	log := logger.GetLogger(c)
	list, err := a.casbinService.GetPolicyPathByAuthorityId(c.Request.Context(), req.AuthorityId)
	if err != nil {
		log.Error("get_casbin_policy_error", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithData(list, c)
}
