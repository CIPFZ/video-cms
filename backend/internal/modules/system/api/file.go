package api

import (
	"fmt"

	"github.com/CIPFZ/video-cms/internal/core/file"
	logger "github.com/CIPFZ/video-cms/internal/core/log"
	"github.com/CIPFZ/video-cms/internal/svc"
	"github.com/CIPFZ/video-cms/pkg/errcode"
	"github.com/CIPFZ/video-cms/pkg/response"
	"github.com/CIPFZ/video-cms/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type FileApi struct {
	svcCtx *svc.ServiceContext
}

func NewFileApi(svcCtx *svc.ServiceContext) *FileApi {
	return &FileApi{svcCtx: svcCtx}
}

func (a *FileApi) Upload(c *gin.Context) {
	_, header, err := c.Request.FormFile("file")
	if err != nil {
		response.FailWithMessage("请选择要上传的文件", c)
		return
	}
	if err := file.ValidateUpload(a.svcCtx.Config.File, header); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	log := logger.GetLogger(c)
	uid := utils.GetUserUUID(c)
	filename := fmt.Sprintf("%s/%s", uid, file.SanitizeUploadName(header.Filename))

	fileURL, _, err := a.svcCtx.OSS.Upload(c.Request.Context(), header, filename)
	if err != nil {
		log.Error("文件上传失败", zap.Error(err))
		response.FailWithError(errcode.FileUploadFailed, c)
		return
	}

	response.OkWithData(gin.H{"url": fileURL}, c)
}
