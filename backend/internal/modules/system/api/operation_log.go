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

// OperationLogApi 提供了操作日志管理的相关接口
type OperationLogApi struct {
	svcCtx       *svc.ServiceContext
	opLogService service.IOperationLogService
}

// NewOperationLogApi 创建一个新的 OperationLogApi 实例
func NewOperationLogApi(svcCtx *svc.ServiceContext, opLogService service.IOperationLogService) *OperationLogApi {
	return &OperationLogApi{
		svcCtx:       svcCtx,
		opLogService: opLogService,
	}
}

// GetOperationLogList 分页获取操作日志列表
// @Tags OperationLog
// @Summary 获取操作日志列表
// @Security ApiKeyAuth
// @Produce application/json
// @Param data body dto.SearchOperationLogReq true "查询和分页参数"
// @Success 200 {object} response.Response{data=dto.PageResult{list=[]model.SysOperationLog}} "成功"
// @Router /operationLog/getOperationLogList [post]
func (a *OperationLogApi) GetOperationLogList(c *gin.Context) {
	log := logger.GetLogger(c)
	var req dto.SearchOperationLogReq
	// 允许空参数，绑定失败也继续执行，使用默认值
	_ = c.ShouldBindJSON(&req)

	req.Normalize()
	list, total, err := a.opLogService.GetOperationLogList(c.Request.Context(), req)
	if err != nil {
		log.Error("get_operation_log_list_error", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}

	response.OkWithDetailed(common.PageResult{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, "获取成功", c)
}

// DeleteOperationLogByIds 批量删除操作日志
// @Tags OperationLog
// @Summary 批量删除操作日志
// @Security ApiKeyAuth
// @Produce application/json
// @Param data body dto.DeleteOperationLogReq true "要删除的日志ID列表"
// @Success 200 {object} response.Response{} "删除成功"
// @Router /operationLog/deleteOperationLogByIds [delete]
func (a *OperationLogApi) DeleteOperationLogByIds(c *gin.Context) {
	log := logger.GetLogger(c)
	var req dto.DeleteOperationLogReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}

	if err := a.opLogService.DeleteOperationLogByIds(c.Request.Context(), req.IDs); err != nil {
		log.Error("delete_operation_log_error", zap.Error(err))
		response.FailWithMessage("删除失败", c)
		return
	}
	response.OkWithMessage("删除成功", c)
}
