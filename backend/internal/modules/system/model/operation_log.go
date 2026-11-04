package model

import (
	"github.com/CIPFZ/video-cms/internal/modules/common"
	"time"
)

// SysOperationLog 操作审计日志
type SysOperationLog struct {
	common.BaseModel

	// --- OTel 链路信息 ---
	TraceID string `json:"traceId" gorm:"type:varchar(64);comment:OpenTelemetry TraceID"`
	SpanID  string `json:"spanId" gorm:"type:varchar(32);comment:OpenTelemetry SpanID"`

	// --- 请求基础信息 ---
	Ip       string        `json:"ip" gorm:"type:varchar(64);comment:请求IP"`
	Method   string        `json:"method" gorm:"type:varchar(10);index;comment:请求方法"`
	Path     string        `json:"path" gorm:"type:varchar(191);index;comment:请求路径"`
	Status   int           `json:"status" gorm:"type:smallint;index;comment:HTTP状态码"`
	Latency  time.Duration `json:"latency" gorm:"type:bigint;comment:耗时(ns)"`
	Agent    string        `json:"agent" gorm:"type:text;comment:User-Agent"`
	ErrorMsg string        `json:"errorMsg" gorm:"column:error_msg;type:text;comment:错误信息"`

	// --- 业务描述 ---
	Module string `json:"module" gorm:"type:varchar(64);comment:所属模块"`
	Remark string `json:"remark" gorm:"type:varchar(128);comment:操作描述"`

	// --- 数据体 (使用 MEDIUMTEXT 防止截断) ---
	Body string `json:"body" gorm:"type:text;comment:请求Body"`
	Resp string `json:"resp" gorm:"type:text;comment:响应Body"`

	// --- 关联用户 ---
	UserID uint    `json:"userId" gorm:"column:user_id;index;comment:用户ID"`
	User   SysUser `json:"user" gorm:"foreignKey:UserID;references:ID"`
}

func (SysOperationLog) TableName() string {
	return "sys_operation_logs"
}
