package model

import "github.com/CIPFZ/video-cms/internal/modules/common"

type SysMenu struct {
	common.BaseModel

	ParentId  uint   `json:"parentId" gorm:"default:0;index;comment:父菜单ID"`
	Path      string `json:"path" gorm:"comment:路由path"`
	Name      string `json:"name" gorm:"comment:路由name(Antd title)"`
	NameEn    string `json:"nameEn" gorm:"column:name_en;default:'';comment:English display name"`
	Component string `json:"component" gorm:"comment:前端组件路径"`
	Access    string `json:"access,omitempty" gorm:"comment:权限标识"` // omitempty 解决 403 问题
	Target    string `json:"target" gorm:"comment:跳转目标"`
	Locale    string `json:"locale" gorm:"comment:国际化"`
	Sort      int    `json:"sort" gorm:"type:int;default:0;index;comment:排序"`
	Icon      string `json:"icon" gorm:"comment:图标"`

	// 数据库列名为 hide_in_menu
	HideInMenu bool `json:"hideInMenu" gorm:"column:hide_in_menu;default:0;comment:是否隐藏"`

	// 子菜单 (虚拟字段，用于前端树形渲染)
	Children []SysMenu `json:"routes" gorm:"-"`
}

func (SysMenu) TableName() string {
	return "sys_menus"
}
