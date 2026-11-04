package model

import "github.com/CIPFZ/video-cms/internal/modules/common"

type JwtBlacklist struct {
	common.BaseModel
	Jwt string `gorm:"type:text;comment:jwt"`
}
