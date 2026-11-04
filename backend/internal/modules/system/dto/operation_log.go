package dto

import (
	"github.com/CIPFZ/video-cms/internal/modules/common"
	"time"
)

// SearchOperationLogReq 操作日志搜索参数
type SearchOperationLogReq struct {
	common.PageInfo
	Method    string     `json:"method" form:"method"`
	Path      string     `json:"path" form:"path"`
	Status    *int       `json:"status" form:"status"`       // 使用指针以允许传 0
	UserID    uint       `json:"userId" form:"userId"`       // 按操作人查询
	Ip        string     `json:"ip" form:"ip"`               // 按IP查询
	TraceID   string     `json:"traceId" form:"traceId"`     // 按链路ID查询
	StartDate *time.Time `json:"startDate" form:"startDate"` // 时间范围搜索
	EndDate   *time.Time `json:"endDate" form:"endDate"`
}

// DeleteOperationLogReq 批量删除参数
type DeleteOperationLogReq struct {
	IDs []uint `json:"ids" binding:"required"`
}
