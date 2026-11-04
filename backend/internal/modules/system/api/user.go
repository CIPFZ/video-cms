package api

import (
	"fmt"
	"net/http"
	"path"
	"time"

	"github.com/CIPFZ/video-cms/internal/core/file"
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

type UserApi struct {
	svcCtx      *svc.ServiceContext
	userService service.IUserService
}

func NewUserApi(svcCtx *svc.ServiceContext, userService service.IUserService) *UserApi {
	return &UserApi{
		svcCtx:      svcCtx,
		userService: userService,
	}
}

func (u *UserApi) Register(c *gin.Context) {
	var req dto.RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}

	user, err := u.userService.Register(c.Request.Context(), req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	user.Password = ""
	response.OkWithData(user, c)
}

func (u *UserApi) Login(c *gin.Context) {
	log := logger.GetLogger(c)
	var req dto.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}

	resp, err := u.userService.Login(c, req)
	if err != nil {
		log.Error("login failed", zap.Error(err))
		response.FailWithMessage("invalid username or password", c)
		return
	}

	now := time.Now().Unix()
	maxAge := int((resp.ExpiresAt / 1000) - now)
	u.setTokenHelper(c, resp.Token, maxAge)

	response.OkWithDetailed(resp, "login successful", c)
}

func (u *UserApi) setTokenHelper(c *gin.Context, token string, maxAge int) {
	var domain string
	if u.svcCtx.Config.System.Environment == "production" {
		domain = ""
	}

	isSecure := u.svcCtx.Config.System.Environment == "production"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("x-token", token, maxAge, "/", domain, isSecure, true)
	c.Header("x-token", token)
	c.Header("new-token", token)
}

func (u *UserApi) SetSelfSetting(c *gin.Context) {}

func (u *UserApi) GetSelfInfo(c *gin.Context) {
	log := logger.GetLogger(c)
	userUUID := utils.GetUserUUID(c)

	user, err := u.userService.GetUserInfo(c.Request.Context(), userUUID)
	if err != nil {
		log.Error("get_user_info_failed", zap.Error(err))
		response.FailWithMessage("获取用户信息失败", c)
		return
	}

	response.OkWithData(user, c)
}

func (u *UserApi) Logout(c *gin.Context) {
	token := c.GetHeader("x-token")
	if token == "" {
		token, _ = c.Cookie("x-token")
	}
	if token == "" {
		response.OkWithMessage("注销成功", c)
		return
	}

	log := logger.GetLogger(c)
	if err := u.userService.Logout(c.Request.Context(), token); err != nil {
		log.Error("logout_api_error", zap.Error(err))
		response.FailWithMessage("注销失败，请重试", c)
		return
	}

	c.SetCookie("x-token", "", -1, "/", "", false, true)
	response.OkWithMessage("注销成功", c)
}

func (u *UserApi) GetUserList(c *gin.Context) {
	var req dto.SearchUserReq
	_ = c.ShouldBindJSON(&req)

	log := logger.GetLogger(c)
	req.Normalize()
	list, total, err := u.userService.GetUserList(c.Request.Context(), req)
	if err != nil {
		log.Error("get_user_list_error", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}

	for i := range list {
		list[i].Password = ""
	}

	response.OkWithDetailed(common.PageResult{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, "获取成功", c)
}

func (u *UserApi) AddUser(c *gin.Context) {
	var req dto.AddUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}

	log := logger.GetLogger(c)
	if err := u.userService.AddUser(c.Request.Context(), req); err != nil {
		log.Error("add_user_error", zap.Error(err))
		response.FailWithMessage("添加失败: "+err.Error(), c)
		return
	}

	response.OkWithMessage("添加成功", c)
}

func (u *UserApi) UpdateUser(c *gin.Context) {
	var req dto.UpdateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}

	log := logger.GetLogger(c)
	if err := u.userService.UpdateUser(c.Request.Context(), req); err != nil {
		log.Error("update_user_error", zap.Error(err))
		response.FailWithMessage("更新失败", c)
		return
	}

	response.OkWithMessage("更新成功", c)
}

func (u *UserApi) DeleteUser(c *gin.Context) {
	var req common.GetByIdReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}

	log := logger.GetLogger(c)
	if err := u.userService.DeleteUser(c.Request.Context(), req.Uint()); err != nil {
		log.Error("delete_user_error", zap.Error(err))
		response.FailWithMessage("删除失败", c)
		return
	}

	response.OkWithMessage("删除成功", c)
}

func (u *UserApi) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}

	log := logger.GetLogger(c)
	if err := u.userService.ResetPassword(c.Request.Context(), req); err != nil {
		log.Error("reset_password_error", zap.Error(err))
		response.FailWithMessage("重置失败", c)
		return
	}

	response.OkWithMessage("重置成功", c)
}

func (u *UserApi) SwitchAuthority(c *gin.Context) {
	type reqBody struct {
		AuthorityID uint `json:"authorityId" binding:"required"`
	}

	var req reqBody
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}

	userUUID := utils.GetUserUUID(c)
	resp, err := u.userService.SwitchAuthority(c.Request.Context(), userUUID, req.AuthorityID)
	if err != nil {
		response.FailWithMessage("切换失败: "+err.Error(), c)
		return
	}

	now := time.Now().Unix()
	maxAge := int((resp.ExpiresAt / 1000) - now)
	u.setTokenHelper(c, resp.Token, maxAge)
	c.Header("new-token", resp.Token)

	response.OkWithData(resp, c)
}

func (u *UserApi) UpdateSelfInfo(c *gin.Context) {
	var req dto.UpdateSelfInfoReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}

	uid := utils.GetUserUUID(c)
	if err := u.userService.UpdateSelfInfo(c.Request.Context(), uid, req); err != nil {
		response.FailWithMessage("更新失败", c)
		return
	}

	response.OkWithMessage("更新成功", c)
}

func (u *UserApi) UpdateUiConfig(c *gin.Context) {
	var req dto.UpdateUiConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithValidation(err, c)
		return
	}

	uid := utils.GetUserUUID(c)
	if err := u.userService.UpdateUiConfig(c.Request.Context(), uid, req); err != nil {
		response.FailWithMessage("配置保存失败", c)
		return
	}

	response.OkWithMessage("保存", c)
}

func (u *UserApi) UploadAvatar(c *gin.Context) {
	log := logger.GetLogger(c)

	_, header, err := c.Request.FormFile("file")
	if err != nil {
		response.FailWithMessage("请选择要上传的图片", c)
		return
	}
	if err := file.ValidateUpload(u.svcCtx.Config.File, header); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	uid := utils.GetUserUUID(c)
	ext := path.Ext(file.SanitizeUploadName(header.Filename))
	if ext == "" {
		ext = ".png"
	}

	fileName := fmt.Sprintf("user/avatar/%s%s", uid, ext)
	avatarURL, _, err := u.svcCtx.OSS.Upload(c.Request.Context(), header, fileName)
	if err != nil {
		log.Error("avatar_upload_failed", zap.Error(err))
		response.FailWithMessage("头像上传失败: "+err.Error(), c)
		return
	}

	if err := u.userService.UpdateAvatar(c.Request.Context(), uid, avatarURL); err != nil {
		log.Error("avatar_update_db_failed", zap.Error(err))
		response.FailWithMessage("更新用户资料失败", c)
		return
	}

	response.OkWithData(gin.H{"url": avatarURL}, c)
}
