package dto

import (
	"time"

	"github.com/CIPFZ/video-cms/internal/modules/common"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
)

type CreateNoticeReq struct {
	Title       string                 `json:"title" binding:"required,min=2,max=128"`
	Content     string                 `json:"content" binding:"required,min=1,max=5000"`
	Level       model.NoticeLevel      `json:"level" binding:"omitempty,oneof=info warning error"`
	TargetType  model.NoticeTargetType `json:"targetType" binding:"required,oneof=all roles users"`
	TargetIDs   []uint                 `json:"targetIds"`
	IsPopup     bool                   `json:"isPopup"`
	NeedConfirm bool                   `json:"needConfirm"`
	StartAt     *time.Time             `json:"startAt"`
	EndAt       *time.Time             `json:"endAt"`
}

type SearchNoticeReq struct {
	common.PageInfo
	Title      string `json:"title" form:"title"`
	Level      string `json:"level" form:"level"`
	TargetType string `json:"targetType" form:"targetType"`
}

type MarkNoticeReadReq struct {
	NoticeID uint `json:"noticeId" binding:"required"`
}

type NoticeListItem struct {
	ID            uint              `json:"ID"`
	CreatedAt     time.Time         `json:"createdAt"`
	Title         string            `json:"title"`
	Content       string            `json:"content"`
	Level         model.NoticeLevel `json:"level"`
	TargetType    string            `json:"targetType"`
	TargetIDs     []uint            `json:"targetIds" gorm:"serializer:json"`
	IsPopup       bool              `json:"isPopup"`
	NeedConfirm   bool              `json:"needConfirm"`
	StartAt       *time.Time        `json:"startAt"`
	EndAt         *time.Time        `json:"endAt"`
	CreatedBy     uint              `json:"createdBy"`
	ReceiverCount int64             `json:"receiverCount"`
	ReadCount     int64             `json:"readCount"`
}

type MyNoticeItem struct {
	ID          uint              `json:"ID"`
	CreatedAt   time.Time         `json:"createdAt"`
	Title       string            `json:"title"`
	Content     string            `json:"content"`
	Level       model.NoticeLevel `json:"level"`
	IsPopup     bool              `json:"isPopup"`
	NeedConfirm bool              `json:"needConfirm"`
	StartAt     *time.Time        `json:"startAt"`
	EndAt       *time.Time        `json:"endAt"`
	ReadAt      *time.Time        `json:"readAt"`
}
