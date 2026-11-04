package model

import (
	"github.com/CIPFZ/video-cms/internal/modules/common"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

const (
	UserActive             = 1
	UserInactive           = 0
	DefaultUserAuthorityID = 888
	DefaultUserAvatar      = "/avatars/capybara.svg"
)

// AvatarPresets 可选头像，文件位于 front-end/public/avatars，与前端 AVATAR_PRESETS 保持一致。
var AvatarPresets = []string{
	"capybara", "bear", "cat", "fox", "rabbit", "panda",
	"dog", "koala", "penguin", "tiger", "pig", "otter",
}

// IsAvatarPreset 判断头像地址是否为预置头像
func IsAvatarPreset(avatar string) bool {
	for _, name := range AvatarPresets {
		if avatar == "/avatars/"+name+".svg" {
			return true
		}
	}
	return false
}

type SysUser struct {
	common.BaseModel // 包含 ID, CreatedAt, UpdatedAt, DeletedAt

	// --- 身份认证 ---
	TokenVersion uint64    `json:"-" gorm:"not null;default:1"`
	UUID         uuid.UUID `json:"uuid" gorm:"type:char(36);index;comment:用户UUID"`
	Username     string    `json:"username" gorm:"type:varchar(64);uniqueIndex;comment:用户名"`
	Password     string    `json:"-" gorm:"type:varchar(128);comment:密码"` // JSON 隐藏

	// --- 个人信息 ---
	NickName string `json:"nickName" gorm:"type:varchar(64);default:系统用户;comment:昵称"`
	Avatar   string `json:"avatar" gorm:"type:varchar(255);default:https://gw.alipayobjects.com/zos/antfincdn/XAosXuNZyF/BiazfanxmamNRoxxVxka.png;comment:头像"`
	Phone    string `json:"phone" gorm:"type:varchar(20);comment:手机号"`
	Email    string `json:"email" gorm:"type:varchar(128);comment:邮箱"`
	Bio      string `json:"bio" gorm:"type:varchar(255);comment:个人简介"`

	// --- 状态与配置 ---
	Status   int            `json:"status" gorm:"type:smallint;default:1;comment:用户状态 1正常 0禁用"`
	Settings datatypes.JSON `json:"settings" gorm:"type:json;comment:个性化设置"`

	// --- 权限关联 ---
	AuthorityID uint         `json:"authorityId" gorm:"default:888;comment:当前角色ID"`
	Authority   SysAuthority `json:"authority" gorm:"belongsTo:SysAuthority;foreignKey:AuthorityID;references:AuthorityId;comment:当前角色"`

	// 多对多关联：用户 <-> 角色
	// 关联表: sys_user_authorities
	// joinForeignKey: 当前表在关联表中的列名 (user_id)
	// joinReferences: 对方表在关联表中的列名 (authority_id)
	Authorities []SysAuthority `json:"authorities" gorm:"many2many:sys_user_authorities;joinForeignKey:user_id;joinReferences:authority_id;"`
}

func (SysUser) TableName() string {
	return "sys_users"
}
