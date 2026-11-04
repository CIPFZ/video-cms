package model

import (
	"time"

	"github.com/CIPFZ/video-cms/internal/modules/common"
)

type NoticeTargetType string

const (
	NoticeTargetAll   NoticeTargetType = "all"
	NoticeTargetRoles NoticeTargetType = "roles"
	NoticeTargetUsers NoticeTargetType = "users"
)

type NoticeLevel string

const (
	NoticeLevelInfo    NoticeLevel = "info"
	NoticeLevelWarning NoticeLevel = "warning"
	NoticeLevelError   NoticeLevel = "error"
)

type SysNotice struct {
	common.BaseModel
	Title       string           `json:"title" gorm:"type:varchar(128);not null;comment:notice title"`
	Content     string           `json:"content" gorm:"type:text;not null;comment:notice content"`
	Level       NoticeLevel      `json:"level" gorm:"type:varchar(16);default:info;comment:notice level"`
	TargetType  NoticeTargetType `json:"targetType" gorm:"type:varchar(16);not null;comment:target type"`
	TargetIDs   []uint           `json:"targetIds" gorm:"serializer:json;type:text"`
	IsPopup     bool             `json:"isPopup" gorm:"default:false;comment:popup on login"`
	NeedConfirm bool             `json:"needConfirm" gorm:"default:false;comment:need manual read confirm"`
	StartAt     *time.Time       `json:"startAt" gorm:"comment:effective start"`
	EndAt       *time.Time       `json:"endAt" gorm:"comment:effective end"`
	CreatedBy   uint             `json:"createdBy" gorm:"index;comment:creator user id"`
}

func (SysNotice) TableName() string {
	return "sys_notices"
}

type SysNoticeReceiver struct {
	common.BaseModel
	NoticeID uint       `json:"noticeId" gorm:"index:idx_notice_user,unique;index;not null;comment:notice id"`
	UserID   uint       `json:"userId" gorm:"index:idx_notice_user,unique;index;not null;comment:receiver user id"`
	ReadAt   *time.Time `json:"readAt" gorm:"comment:read timestamp"`
}

func (SysNoticeReceiver) TableName() string {
	return "sys_notice_receivers"
}
