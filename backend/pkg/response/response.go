package response

import (
	"errors"
	"net/http"

	"github.com/CIPFZ/video-cms/pkg/errcode"
	"github.com/gin-gonic/gin"
)

// Response 基础响应结构
type Response struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

// PageResult 分页数据结构
type PageResult struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}

// result 统一的内部响应处理
func result(c *gin.Context, code int, msg string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: code,
		Msg:  LocalizeMessage(c, msg),
		Data: data,
	})
}

// -------------------------------------------------------------------------
// ✅ 成功响应类
// -------------------------------------------------------------------------

func Ok(c *gin.Context) {
	result(c, errcode.Success.Code, errcode.Success.Msg, nil)
}

func OkWithMessage(message string, c *gin.Context) {
	result(c, errcode.Success.Code, message, nil)
}

func OkWithData(data interface{}, c *gin.Context) {
	result(c, errcode.Success.Code, errcode.Success.Msg, data)
}

func OkWithDetailed(data interface{}, message string, c *gin.Context) {
	result(c, errcode.Success.Code, message, data)
}

// OkWithPage ✨ 分页响应快捷方法 ✨
func OkWithPage(list interface{}, total int64, page, pageSize int, c *gin.Context) {
	result(c, errcode.Success.Code, errcode.Success.Msg, PageResult{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// -------------------------------------------------------------------------
// ❌ 错误响应类
// -------------------------------------------------------------------------

// FailWithError ✨ 智能错误处理 (最推荐使用) ✨
// 自动识别错误类型：
// 1. 如果是业务错误 (*errcode.Error)，返回对应 Code 和 Msg
// 2. 如果是普通错误 (error)，包装为 ServerError 并附带详情
func FailWithError(err error, c *gin.Context) {
	if err == nil {
		Ok(c)
		return
	}

	// 尝试强转为自定义业务错误
	var e *errcode.Error
	if errors.As(err, &e) {
		result(c, e.Code, e.Msg, nil)
		return
	}

	// 默认为系统未知错误
	// 生产环境为了安全，这里可以选择不返回 err.Error()，或者只返回 "系统内部错误"
	// 这里为了开发方便，我们将详细错误带上
	sysErr := errcode.ServerError.WithDetails(err.Error())
	result(c, sysErr.Code, sysErr.Msg, nil)
}

// FailWithCode 手动指定错误码
func FailWithCode(err *errcode.Error, c *gin.Context) {
	result(c, err.Code, err.Msg, nil)
}

// FailWithMessage 自定义消息失败 (使用默认 ServerError 码)
func FailWithMessage(message string, c *gin.Context) {
	result(c, errcode.ServerError.Code, message, nil)
}
