package dto

import "github.com/CIPFZ/video-cms/internal/modules/common"

// SearchApiReq API分页查询
type SearchApiReq struct {
	Path        string `json:"path"`        // 路径关键字
	Description string `json:"description"` // 描述关键字
	Method      string `json:"method"`      // 方法关键字
	ApiGroup    string `json:"apiGroup"`    // 分组关键字
	common.PageInfo
}

// CreateApiReq 新增 API
type CreateApiReq struct {
	Path        string `json:"path" binding:"required"`
	Description string `json:"description" binding:"required"`
	ApiGroup    string `json:"apiGroup" binding:"required"`
	Method      string `json:"method" binding:"required"`
}

// UpdateApiReq 更新 API
type UpdateApiReq struct {
	ID          uint   `json:"id" binding:"required"`
	Path        string `json:"path" binding:"required"`
	Description string `json:"description" binding:"required"`
	ApiGroup    string `json:"apiGroup" binding:"required"`
	Method      string `json:"method" binding:"required"`
}

// DeleteApiReq 删除/批量删除 API
type DeleteApiReq struct {
	ID  uint   `json:"id"`  // 单个删除用
	IDs []uint `json:"ids"` // 批量删除用
}
