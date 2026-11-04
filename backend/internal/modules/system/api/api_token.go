package api

import (
	logger "github.com/CIPFZ/video-cms/internal/core/log"
	tokenCore "github.com/CIPFZ/video-cms/internal/core/token"
	"github.com/CIPFZ/video-cms/internal/middleware"
	"github.com/CIPFZ/video-cms/internal/modules/system/dto"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/CIPFZ/video-cms/internal/modules/system/service"
	"github.com/CIPFZ/video-cms/internal/svc"
	"github.com/CIPFZ/video-cms/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ApiTokenApi struct {
	svcCtx          *svc.ServiceContext
	apiTokenService service.IApiTokenService
}

func NewApiTokenApi(svcCtx *svc.ServiceContext, apiTokenService service.IApiTokenService) *ApiTokenApi {
	return &ApiTokenApi{
		svcCtx:          svcCtx,
		apiTokenService: apiTokenService,
	}
}

func (a *ApiTokenApi) GetApiTokenList(c *gin.Context) {
	var req dto.SearchApiTokenReq
	if err := c.ShouldBind(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}

	req.Normalize()
	list, total, err := a.apiTokenService.GetApiTokenList(c.Request.Context(), req)
	if err != nil {
		logger.GetLogger(c).Error("get_api_token_list_error", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithPage(list, total, req.Page, req.PageSize, c)
}

func (a *ApiTokenApi) CreateApiToken(c *gin.Context) {
	var req dto.CreateApiTokenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}

	createdBy := c.GetUint(middleware.CtxKeyUserID)
	resp, err := a.apiTokenService.CreateApiToken(c.Request.Context(), createdBy, req)
	if err != nil {
		logger.GetLogger(c).Error("create_api_token_error", zap.Error(err))
		response.FailWithMessage("创建失败: "+err.Error(), c)
		return
	}
	response.OkWithData(resp, c)
}

func (a *ApiTokenApi) GetApiTokenDetail(c *gin.Context) {
	var req dto.ApiTokenDetailReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}
	resp, err := a.apiTokenService.GetApiTokenDetail(c.Request.Context(), req.ID)
	if err != nil {
		logger.GetLogger(c).Error("get_api_token_detail_error", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithData(resp, c)
}

func (a *ApiTokenApi) UpdateApiToken(c *gin.Context) {
	var req dto.UpdateApiTokenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}
	if err := a.apiTokenService.UpdateApiToken(c.Request.Context(), req); err != nil {
		logger.GetLogger(c).Error("update_api_token_error", zap.Error(err))
		response.FailWithMessage("更新失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("更新成功", c)
}

func (a *ApiTokenApi) DeleteApiToken(c *gin.Context) {
	var req dto.DeleteApiTokenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}
	if err := a.apiTokenService.DeleteApiToken(c.Request.Context(), req); err != nil {
		logger.GetLogger(c).Error("delete_api_token_error", zap.Error(err))
		response.FailWithMessage("删除失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

func (a *ApiTokenApi) ResetApiToken(c *gin.Context) {
	var req dto.ToggleApiTokenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}
	resp, err := a.apiTokenService.ResetApiToken(c.Request.Context(), req.ID)
	if err != nil {
		logger.GetLogger(c).Error("reset_api_token_error", zap.Error(err))
		response.FailWithMessage("重置失败: "+err.Error(), c)
		return
	}
	response.OkWithData(resp, c)
}

func (a *ApiTokenApi) EnableApiToken(c *gin.Context) {
	a.toggleApiToken(c, true)
}

func (a *ApiTokenApi) DisableApiToken(c *gin.Context) {
	a.toggleApiToken(c, false)
}

func (a *ApiTokenApi) toggleApiToken(c *gin.Context, enabled bool) {
	var req dto.ToggleApiTokenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}

	var err error
	if enabled {
		err = a.apiTokenService.EnableApiToken(c.Request.Context(), req.ID)
	} else {
		err = a.apiTokenService.DisableApiToken(c.Request.Context(), req.ID)
	}
	if err != nil {
		logger.GetLogger(c).Error("toggle_api_token_error", zap.Bool("enabled", enabled), zap.Error(err))
		response.FailWithMessage("操作失败: "+err.Error(), c)
		return
	}
	response.Ok(c)
}

func (a *ApiTokenApi) GetOptions(c *gin.Context) {
	var apis []model.SysApi
	query := a.svcCtx.DB.WithContext(c.Request.Context()).Where("1 = 0")
	for path, method := range tokenCore.Endpoints(a.svcCtx.Config.System.RouterPrefix) {
		query = query.Or("path = ? AND method = ?", path, method)
	}
	if err := query.Find(&apis).Error; err != nil {
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithPage(apis, int64(len(apis)), 1, 100, c)
}
