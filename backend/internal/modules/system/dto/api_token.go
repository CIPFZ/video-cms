package dto

import "github.com/CIPFZ/video-cms/internal/modules/common"

type SearchApiTokenReq struct {
	Name    string `json:"name" form:"name"`
	Enabled *bool  `json:"enabled" form:"enabled"`
	common.PageInfo
}

type CreateApiTokenReq struct {
	Name           string  `json:"name" binding:"required,max=100"`
	Description    string  `json:"description" binding:"max=255"`
	ExpiresAt      *string `json:"expiresAt"`
	NeverExpire    bool    `json:"neverExpire"`
	MaxConcurrency int     `json:"maxConcurrency" binding:"required,min=1,max=1000"`
	ApiIds         []uint  `json:"apiIds" binding:"required"`
}

type UpdateApiTokenReq struct {
	ID             uint    `json:"id" binding:"required"`
	Name           string  `json:"name" binding:"required,max=100"`
	Description    string  `json:"description" binding:"max=255"`
	ExpiresAt      *string `json:"expiresAt"`
	NeverExpire    bool    `json:"neverExpire"`
	MaxConcurrency int     `json:"maxConcurrency" binding:"required,min=1,max=1000"`
	ApiIds         []uint  `json:"apiIds" binding:"required"`
}

type DeleteApiTokenReq struct {
	ID  uint   `json:"id"`
	IDs []uint `json:"ids"`
}

type ToggleApiTokenReq struct {
	ID uint `json:"id" binding:"required"`
}

type ApiTokenDetailReq struct {
	ID uint `json:"id" form:"id" binding:"required"`
}

type ApiTokenResponse struct {
	ID             uint            `json:"ID"`
	TokenPrefix    string          `json:"tokenPrefix"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	ExpiresAt      *string         `json:"expiresAt"`
	NeverExpire    bool            `json:"neverExpire"`
	MaxConcurrency int             `json:"maxConcurrency"`
	Enabled        bool            `json:"enabled"`
	LastUsedAt     *string         `json:"lastUsedAt"`
	CreatedAt      string          `json:"createdAt"`
	CreatedBy      uint            `json:"createdBy"`
	Apis           []ApiSimpleItem `json:"apis"`
}

type ApiTokenSecretResponse struct {
	ApiTokenResponse
	Token string `json:"token"`
}

type ApiSimpleItem struct {
	ID          uint   `json:"ID"`
	Path        string `json:"path"`
	Method      string `json:"method"`
	ApiGroup    string `json:"apiGroup"`
	Description string `json:"description"`
}
