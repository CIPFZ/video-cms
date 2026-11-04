package model

import (
	"time"

	"github.com/CIPFZ/video-cms/internal/modules/common"
)

type SysApiToken struct {
	common.BaseModel

	TokenHash      string     `json:"-" gorm:"type:varchar(64);uniqueIndex;comment:token hash"`
	TokenPrefix    string     `json:"tokenPrefix" gorm:"type:varchar(16);comment:token prefix"`
	Name           string     `json:"name" gorm:"type:varchar(100);comment:token name"`
	Description    string     `json:"description" gorm:"type:varchar(255);comment:token description"`
	ExpiresAt      *time.Time `json:"expiresAt" gorm:"comment:expires at"`
	MaxConcurrency int        `json:"maxConcurrency" gorm:"default:5;comment:max concurrency"`
	Enabled        bool       `json:"enabled" gorm:"default:true;comment:enabled"`
	LastUsedAt     *time.Time `json:"lastUsedAt" gorm:"comment:last used at"`
	CreatedBy      uint       `json:"createdBy" gorm:"comment:created by"`

	Apis []SysApi `json:"apis" gorm:"many2many:sys_api_token_apis;joinForeignKey:ApiTokenId;joinReferences:ApiId"`
}

func (SysApiToken) TableName() string {
	return "sys_api_tokens"
}

type SysApiTokenApi struct {
	ApiTokenId uint `gorm:"column:api_token_id;primaryKey;comment:api token id"`
	ApiId      uint `gorm:"column:api_id;primaryKey;comment:api id"`
}

func (SysApiTokenApi) TableName() string {
	return "sys_api_token_apis"
}
