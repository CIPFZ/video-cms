package model

import "github.com/CIPFZ/video-cms/internal/modules/common"

// SysApi 系统API表
type SysApi struct {
	common.BaseModel
	Path        string `json:"path" gorm:"comment:api路径"`             // e.g. /api/v1/user/list
	Description string `json:"description" gorm:"comment:api中文描述"`    // e.g. 获取用户列表
	ApiGroup    string `json:"apiGroup" gorm:"comment:api组"`          // e.g. 用户管理
	Method      string `json:"method" gorm:"default:POST;comment:方法"` // e.g. POST, GET
}

func (SysApi) TableName() string {
	return "sys_apis"
}
