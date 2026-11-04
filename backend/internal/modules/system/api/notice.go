package api

import (
	logger "github.com/CIPFZ/video-cms/internal/core/log"
	"github.com/CIPFZ/video-cms/internal/modules/common"
	"github.com/CIPFZ/video-cms/internal/modules/system/dto"
	"github.com/CIPFZ/video-cms/internal/modules/system/service"
	"github.com/CIPFZ/video-cms/internal/svc"
	"github.com/CIPFZ/video-cms/pkg/response"
	"github.com/CIPFZ/video-cms/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type NoticeApi struct {
	svcCtx        *svc.ServiceContext
	noticeService service.INoticeService
}

func NewNoticeApi(svcCtx *svc.ServiceContext, noticeService service.INoticeService) *NoticeApi {
	return &NoticeApi{svcCtx: svcCtx, noticeService: noticeService}
}

func (a *NoticeApi) CreateNotice(c *gin.Context) {
	var req dto.CreateNoticeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}
	if err := a.noticeService.CreateNotice(c.Request.Context(), req, utils.GetUserID(c)); err != nil {
		response.FailWithMessage("create notice failed: "+err.Error(), c)
		return
	}
	response.OkWithMessage("created", c)
}

func (a *NoticeApi) GetNoticeList(c *gin.Context) {
	var req dto.SearchNoticeReq
	if err := c.ShouldBind(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}
	req.Normalize()
	list, total, err := a.noticeService.GetNoticeList(c.Request.Context(), req)
	if err != nil {
		logger.GetLogger(c).Error("get_notice_list_failed", zap.Error(err))
		response.FailWithMessage("get notice list failed", c)
		return
	}
	response.OkWithDetailed(common.PageResult{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, "ok", c)
}

func (a *NoticeApi) GetMyNotices(c *gin.Context) {
	var q struct {
		common.PageInfo
		PopupOnly bool `form:"popupOnly"`
	}
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithValidation(err, c)
		return
	}
	q.Normalize()
	list, total, err := a.noticeService.GetMyNotices(c.Request.Context(), utils.GetUserID(c), q.Page, q.PageSize, q.PopupOnly)
	if err != nil {
		logger.GetLogger(c).Error("get_my_notices_failed", zap.Error(err))
		response.FailWithMessage("get my notices failed", c)
		return
	}
	response.OkWithDetailed(common.PageResult{
		List:     list,
		Total:    total,
		Page:     q.Page,
		PageSize: q.PageSize,
	}, "ok", c)
}

func (a *NoticeApi) MarkRead(c *gin.Context) {
	var req dto.MarkNoticeReadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}
	if err := a.noticeService.MarkRead(c.Request.Context(), req.NoticeID, utils.GetUserID(c)); err != nil {
		response.FailWithMessage("mark read failed: "+err.Error(), c)
		return
	}
	response.OkWithMessage("ok", c)
}
