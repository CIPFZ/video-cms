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

// SysApiApi 提供了系统 API 管理的相关接口
type SysApiApi struct {
	svcCtx     *svc.ServiceContext
	apiService service.IApiService
}

// NewSysApiApi 创建一个新的 SysApiApi 实例
func NewSysApiApi(svcCtx *svc.ServiceContext, apiService service.IApiService) *SysApiApi {
	return &SysApiApi{
		svcCtx:     svcCtx,
		apiService: apiService,
	}
}

// GetApiList 分页获取 API 列表
// @Tags SysApi
// @Summary 分页获取API列表
// @Security ApiKeyAuth
// @Produce application/json
// @Param data query dto.SearchApiReq true "查询参数"
// @Success 200 {object} response.Response{data=dto.PageResult{list=[]model.SysApi}} "成功"
// @Router /api/getApiList [get]
func (a *SysApiApi) GetApiList(c *gin.Context) {
	var req dto.SearchApiReq
	// ShouldBind 支持从 JSON 或 Query 参数绑定
	if err := c.ShouldBind(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}

	log := logger.GetLogger(c)
	req.Normalize()
	list, total, err := a.apiService.GetApiList(c.Request.Context(), req)
	if err != nil {
		log.Error("get_api_list_error", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithPage(list, total, req.Page, req.PageSize, c)
}

// CreateApi 创建新的 API
// @Tags SysApi
// @Summary 创建API
// @Security ApiKeyAuth
// @Produce application/json
// @Param data body dto.CreateApiReq true "API信息"
// @Success 200 {object} response.Response{} "成功"
// @Router /api/createApi [post]
func (a *SysApiApi) CreateApi(c *gin.Context) {
	var req dto.CreateApiReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}
	log := logger.GetLogger(c)
	if err := a.apiService.CreateApi(c.Request.Context(), req); err != nil {
		log.Error("create_api_error", zap.Error(err))
		response.FailWithMessage("创建失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("创建成功", c)
}

// UpdateApi 更新指定的 API
// @Tags SysApi
// @Summary 更新API
// @Security ApiKeyAuth
// @Produce application/json
// @Param data body dto.UpdateApiReq true "API信息"
// @Success 200 {object} response.Response{} "成功"
// @Router /api/updateApi [put]
func (a *SysApiApi) UpdateApi(c *gin.Context) {
	var req dto.UpdateApiReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}
	log := logger.GetLogger(c)
	if err := a.apiService.UpdateApi(c.Request.Context(), req); err != nil {
		log.Error("update_api_error", zap.Error(err))
		response.FailWithMessage("更新失败: "+err.Error(), c)
		return
	}

	response.OkWithMessage("更新成功", c)
}

// DeleteApi 删除一个或多个 API
// @Tags SysApi
// @Summary 删除API
// @Security ApiKeyAuth
// @Produce application/json
// @Param data body dto.DeleteApiReq true "API ID"
// @Success 200 {object} response.Response{} "成功"
// @Router /api/deleteApi [delete]
func (a *SysApiApi) DeleteApi(c *gin.Context) {
	var req dto.DeleteApiReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}
	log := logger.GetLogger(c)
	if err := a.apiService.DeleteApi(c.Request.Context(), req); err != nil {
		log.Error("delete_api_error", zap.Error(err))
		response.FailWithMessage("删除失败: "+err.Error(), c)
		return
	}

	response.OkWithMessage("删除成功", c)
}
