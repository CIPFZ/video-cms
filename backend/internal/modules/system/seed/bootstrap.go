package seed

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/CIPFZ/video-cms/internal/core/config"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/CIPFZ/video-cms/pkg/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	defaultSeedAdminEnabled      = true
	defaultSeedAdminUsername     = "admin"
	defaultSeedAdminPassword     = "Admin@123456"
	defaultSeedAdminAuthorityID  = uint(1)
	defaultSeedAdminDefaultRoute = "dashboard/workplace"
)

type seedAdminOptions struct {
	Enabled      bool
	Username     string
	Password     string
	AuthorityID  uint
	DefaultRoute string
}

type seedMenu struct {
	Key        string
	ParentKey  string
	Path       string
	Name       string
	Component  string
	Icon       string
	Locale     string
	Access     string
	Target     string
	Sort       int
	HideInMenu bool
}

type seedApi struct {
	Path        string
	Method      string
	ApiGroup    string
	Description string
}

type casbinRuleSeed struct {
	Ptype string `gorm:"column:ptype"`
	V0    string `gorm:"column:v0"`
	V1    string `gorm:"column:v1"`
	V2    string `gorm:"column:v2"`
	V3    string `gorm:"column:v3"`
	V4    string `gorm:"column:v4"`
	V5    string `gorm:"column:v5"`
}

func (casbinRuleSeed) TableName() string {
	return "sys_casbin_rules"
}

func Bootstrap(ctx context.Context, database *gorm.DB, cfg *config.Config, logger *zap.Logger) error {
	var count int64
	if err := database.Unscoped().Model(&model.SysUser{}).Count(&count).Error; err != nil {
		return err
	}
	// Existing installations own their users, menus and grants. Never reseed them.
	if count != 0 {
		return nil
	}
	opts := loadSeedAdminOptions(cfg)
	if !opts.Enabled {
		logger.Info("seed admin disabled")
		return nil
	}

	return database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureAuthorities(tx, opts); err != nil {
			return err
		}

		menuIDs, err := ensureBaseMenus(tx)
		if err != nil {
			return err
		}
		if err := bindAuthorityMenus(tx, menuIDs, opts.AuthorityID); err != nil {
			return err
		}

		apiIDs, err := ensureBaseApis(tx)
		if err != nil {
			return err
		}
		if err := bindAuthorityApis(tx, apiIDs, opts.AuthorityID); err != nil {
			return err
		}

		if err := ensureCasbinPolicies(tx, apiIDs, opts.AuthorityID); err != nil {
			return err
		}

		if err := EnsureTokenEndpoints(tx, cfg.System.RouterPrefix); err != nil {
			return err
		}

		if err := ensureAdminUser(tx, opts); err != nil {
			return err
		}

		logger.Info("seed base system data finished",
			zap.String("username", opts.Username),
			zap.Uint("authorityId", opts.AuthorityID),
		)
		return nil
	})
}

func loadSeedAdminOptions(cfg *config.Config) seedAdminOptions {
	env := strings.ToLower(strings.TrimSpace(cfg.System.Environment))
	isDevLike := env == "dev" || env == "development" || env == "local"

	opts := seedAdminOptions{
		Enabled:      defaultSeedAdminEnabled && isDevLike,
		Username:     defaultSeedAdminUsername,
		Password:     defaultSeedAdminPassword,
		AuthorityID:  defaultSeedAdminAuthorityID,
		DefaultRoute: defaultSeedAdminDefaultRoute,
	}

	if v := strings.TrimSpace(os.Getenv("SEED_ADMIN_ENABLED")); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			opts.Enabled = parsed
		}
	}
	if v := strings.TrimSpace(os.Getenv("SEED_ADMIN_USERNAME")); v != "" {
		opts.Username = v
	}
	if v := strings.TrimSpace(os.Getenv("SEED_ADMIN_PASSWORD")); v != "" {
		opts.Password = v
	}
	if v := strings.TrimSpace(os.Getenv("SEED_ADMIN_AUTHORITY_ID")); v != "" {
		if parsed, err := strconv.ParseUint(v, 10, 32); err == nil {
			opts.AuthorityID = uint(parsed)
		}
	}
	if v := strings.TrimSpace(os.Getenv("SEED_ADMIN_DEFAULT_ROUTER")); v != "" {
		opts.DefaultRoute = strings.Trim(v, "/")
	}
	return opts
}

func ensureAuthorities(tx *gorm.DB, opts seedAdminOptions) error {
	authorities := []model.SysAuthority{
		{AuthorityId: opts.AuthorityID, AuthorityName: "Administrator", ParentId: 0, DefaultRouter: opts.DefaultRoute},
		{AuthorityId: 888, AuthorityName: "CommonUser", ParentId: 0, DefaultRouter: "dashboard/workplace"},
		{AuthorityId: 9528, AuthorityName: "TestUser", ParentId: 0, DefaultRouter: "dashboard/workplace"},
		{AuthorityId: 8881, AuthorityName: "CommonUserChild", ParentId: 888, DefaultRouter: "dashboard/workplace"},
	}

	seen := make(map[uint]struct{}, len(authorities))
	for _, auth := range authorities {
		if _, ok := seen[auth.AuthorityId]; ok {
			continue
		}
		seen[auth.AuthorityId] = struct{}{}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&auth).Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureBaseMenus(tx *gorm.DB) (map[string]uint, error) {
	menus := []seedMenu{
		{Key: "dashboard", Path: "/dashboard/workplace", Name: "menu.dashboard.workplace", Component: "dashboard/workplace", Icon: "DashboardOutlined", Locale: "menu.dashboard.workplace"},
		{Key: "state", Path: "/state", Name: "menu.state", Component: "state", Icon: "CloudServerOutlined", Locale: "menu.state"},
		{Key: "about", Path: "/about", Name: "menu.about", Component: "about", Icon: "InfoCircleOutlined", Locale: "menu.about"},
		{Key: "sys_root", Path: "/sys", Name: "menu.system", Component: "components/RouterLayout", Icon: "SettingOutlined", Locale: "menu.system"},
		{Key: "sys_user", ParentKey: "sys_root", Path: "/sys/user", Name: "menu.system.user", Component: "sys/user", Icon: "UserOutlined", Locale: "menu.system.user"},
		{Key: "sys_authority", ParentKey: "sys_root", Path: "/sys/authority", Name: "menu.system.authority", Component: "sys/authority", Icon: "TeamOutlined", Locale: "menu.system.authority"},
		{Key: "sys_menu", ParentKey: "sys_root", Path: "/sys/menu", Name: "menu.system.menu", Component: "sys/menu", Icon: "MenuOutlined", Locale: "menu.system.menu"},
		{Key: "sys_api", ParentKey: "sys_root", Path: "/sys/api", Name: "menu.system.api", Component: "sys/api", Icon: "ApiOutlined", Locale: "menu.system.api"},
		{Key: "sys_api_token", ParentKey: "sys_root", Path: "/sys/api-token", Name: "menu.system.apiToken", Component: "sys/api-token", Icon: "KeyOutlined", Locale: "menu.system.apiToken"},
		{Key: "sys_operation", ParentKey: "sys_root", Path: "/sys/operation", Name: "menu.system.operation", Component: "sys/operation", Icon: "HistoryOutlined", Locale: "menu.system.operation"},
		{Key: "sys_notice", ParentKey: "sys_root", Path: "/sys/notice", Name: "menu.system.notice", Component: "sys/notice", Icon: "NotificationOutlined", Locale: "menu.system.notice"},
		{Key: "account_settings", Path: "/account/settings", Name: "menu.account.settings", Component: "user/info", Icon: "ProfileOutlined", Locale: "menu.account.settings", HideInMenu: true},
	}

	menuIDs := make(map[string]uint, len(menus))
	for _, item := range menus {
		if name, ok := MenuNames[item.Locale]; ok {
			item.Name = name
		}
		item.Sort = MenuOrder[item.Path]
		parentID := uint(0)
		if item.ParentKey != "" {
			pid, ok := menuIDs[item.ParentKey]
			if !ok {
				return nil, errors.New("seed menu parent not found: " + item.ParentKey)
			}
			parentID = pid
		}

		var menu model.SysMenu
		err := tx.Where("path = ?", item.Path).First(&menu).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			menu = model.SysMenu{
				ParentId:   parentID,
				Path:       item.Path,
				Name:       item.Name,
				NameEn:     MenuNamesEn[item.Locale],
				Component:  item.Component,
				Access:     item.Access,
				Target:     item.Target,
				Locale:     item.Locale,
				Sort:       item.Sort,
				Icon:       item.Icon,
				HideInMenu: item.HideInMenu,
			}
			if err := tx.Create(&menu).Error; err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}
		menuIDs[item.Key] = menu.ID
	}
	return menuIDs, nil
}

func bindAuthorityMenus(tx *gorm.DB, menuIDs map[string]uint, adminAuthorityID uint) error {
	roleMenuKeys := map[uint][]string{
		adminAuthorityID: {"dashboard", "state", "about", "sys_root", "sys_user", "sys_authority", "sys_menu", "sys_api", "sys_api_token", "sys_operation", "sys_notice", "account_settings"},
		9528:             {"dashboard", "state", "about", "sys_root", "sys_user", "sys_authority", "sys_menu", "sys_api", "sys_api_token", "sys_operation", "sys_notice", "account_settings"},
		888:              {"dashboard", "state", "about", "account_settings"},
		8881:             {"dashboard", "about", "account_settings"},
	}

	relations := make([]model.SysAuthorityMenu, 0)
	for authorityID, keys := range roleMenuKeys {
		for _, key := range keys {
			menuID, ok := menuIDs[key]
			if !ok {
				continue
			}
			relations = append(relations, model.SysAuthorityMenu{AuthorityId: authorityID, MenuId: menuID})
		}
	}
	if len(relations) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&relations).Error
}

func ensureBaseApis(tx *gorm.DB) (map[string]uint, error) {
	apis := []seedApi{
		{Path: "/api/v1/sys/user/getSelfInfo", Method: "GET", ApiGroup: "system-user", Description: "Get current user"},
		{Path: "/api/v1/sys/user/getUserList", Method: "POST", ApiGroup: "system-user", Description: "Get user list"},
		{Path: "/api/v1/sys/user/logout", Method: "POST", ApiGroup: "system-user", Description: "Logout"},
		{Path: "/api/v1/sys/user/info", Method: "PUT", ApiGroup: "system-user", Description: "Update self info"},
		{Path: "/api/v1/sys/user/ui-config", Method: "PUT", ApiGroup: "system-user", Description: "Update UI config"},
		{Path: "/api/v1/sys/user/avatar", Method: "POST", ApiGroup: "system-user", Description: "Upload avatar"},
		{Path: "/api/v1/sys/user/switchAuthority", Method: "POST", ApiGroup: "system-user", Description: "Switch authority"},
		{Path: "/api/v1/sys/user/addUser", Method: "POST", ApiGroup: "system-user", Description: "Add user"},
		{Path: "/api/v1/sys/user/updateUser", Method: "PUT", ApiGroup: "system-user", Description: "Update user"},
		{Path: "/api/v1/sys/user/deleteUser", Method: "DELETE", ApiGroup: "system-user", Description: "Delete user"},
		{Path: "/api/v1/sys/user/resetPassword", Method: "POST", ApiGroup: "system-user", Description: "Reset password"},

		{Path: "/api/v1/sys/menu/getMenu", Method: "GET", ApiGroup: "system-menu", Description: "Get current menu"},
		{Path: "/api/v1/sys/menu/getMenuList", Method: "POST", ApiGroup: "system-menu", Description: "Get menu list"},
		{Path: "/api/v1/sys/menu/getMenuAuthority", Method: "POST", ApiGroup: "system-menu", Description: "Get authority menus"},
		{Path: "/api/v1/sys/menu/addBaseMenu", Method: "POST", ApiGroup: "system-menu", Description: "Create menu"},
		{Path: "/api/v1/sys/menu/updateBaseMenu", Method: "PUT", ApiGroup: "system-menu", Description: "Update menu"},
		{Path: "/api/v1/sys/menu/deleteBaseMenu", Method: "DELETE", ApiGroup: "system-menu", Description: "Delete menu"},

		{Path: "/api/v1/sys/authority/getAuthorityList", Method: "POST", ApiGroup: "system-authority", Description: "Get authority list"},
		{Path: "/api/v1/sys/authority/createAuthority", Method: "POST", ApiGroup: "system-authority", Description: "Create authority"},
		{Path: "/api/v1/sys/authority/updateAuthority", Method: "PUT", ApiGroup: "system-authority", Description: "Update authority"},
		{Path: "/api/v1/sys/authority/deleteAuthority", Method: "DELETE", ApiGroup: "system-authority", Description: "Delete authority"},
		{Path: "/api/v1/sys/authority/setAuthorityMenus", Method: "POST", ApiGroup: "system-authority", Description: "Set authority menus"},

		{Path: "/api/v1/sys/api/getApiList", Method: "POST", ApiGroup: "system-api", Description: "Get API list"},
		{Path: "/api/v1/sys/api/createApi", Method: "POST", ApiGroup: "system-api", Description: "Create API"},
		{Path: "/api/v1/sys/api/updateApi", Method: "PUT", ApiGroup: "system-api", Description: "Update API"},
		{Path: "/api/v1/sys/api/deleteApi", Method: "DELETE", ApiGroup: "system-api", Description: "Delete API"},

		{Path: "/api/v1/sys/api-token/getApiTokenList", Method: "POST", ApiGroup: "system-api-token", Description: "Get API token list"},
		{Path: "/api/v1/sys/api-token/detail", Method: "GET", ApiGroup: "system-api-token", Description: "Get API token detail"},
		{Path: "/api/v1/sys/api-token/create", Method: "POST", ApiGroup: "system-api-token", Description: "Create API token"},
		{Path: "/api/v1/sys/api-token/update", Method: "PUT", ApiGroup: "system-api-token", Description: "Update API token"},
		{Path: "/api/v1/sys/api-token/delete", Method: "DELETE", ApiGroup: "system-api-token", Description: "Delete API token"},
		{Path: "/api/v1/sys/api-token/reset", Method: "POST", ApiGroup: "system-api-token", Description: "Reset API token"},
		{Path: "/api/v1/sys/api-token/enable", Method: "POST", ApiGroup: "system-api-token", Description: "Enable API token"},
		{Path: "/api/v1/sys/api-token/disable", Method: "POST", ApiGroup: "system-api-token", Description: "Disable API token"},

		{Path: "/api/v1/sys/casbin/getPolicyPathByAuthorityId", Method: "POST", ApiGroup: "system-casbin", Description: "Get casbin policy list"},
		{Path: "/api/v1/sys/casbin/updateCasbin", Method: "POST", ApiGroup: "system-casbin", Description: "Update casbin policy"},

		{Path: "/api/v1/sys/operationLog/getOperationLogList", Method: "POST", ApiGroup: "system-operation", Description: "Get operation logs"},
		{Path: "/api/v1/sys/operationLog/deleteOperationLogByIds", Method: "DELETE", ApiGroup: "system-operation", Description: "Delete operation logs"},
		{Path: "/api/v1/sys/file/upload", Method: "POST", ApiGroup: "system-file", Description: "Upload file"},
		{Path: "/api/v1/sys/system/getServerInfo", Method: "POST", ApiGroup: "system-state", Description: "Get server state"},
		{Path: "/api/v1/sys/notice/createNotice", Method: "POST", ApiGroup: "system-notice", Description: "Create notice"},
		{Path: "/api/v1/sys/notice/getNoticeList", Method: "POST", ApiGroup: "system-notice", Description: "Get notice list"},
		{Path: "/api/v1/sys/notice/getMyNotices", Method: "GET", ApiGroup: "system-notice", Description: "Get my notices"},
		{Path: "/api/v1/sys/notice/markRead", Method: "POST", ApiGroup: "system-notice", Description: "Mark notice as read"},
	}

	apiIDs := make(map[string]uint, len(apis))
	for _, item := range apis {
		sign := apiSign(item.Method, item.Path)

		var api model.SysApi
		err := tx.Where("path = ? AND method = ?", item.Path, item.Method).First(&api).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api = model.SysApi{Path: item.Path, Method: item.Method, ApiGroup: item.ApiGroup, Description: item.Description}
			if err := tx.Create(&api).Error; err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}
		apiIDs[sign] = api.ID
	}
	return apiIDs, nil
}

func bindAuthorityApis(tx *gorm.DB, apiIDs map[string]uint, adminAuthorityID uint) error {
	fullAccess := []string{
		apiSign("GET", "/api/v1/sys/user/getSelfInfo"),
		apiSign("POST", "/api/v1/sys/user/getUserList"),
		apiSign("POST", "/api/v1/sys/user/logout"),
		apiSign("PUT", "/api/v1/sys/user/info"),
		apiSign("PUT", "/api/v1/sys/user/ui-config"),
		apiSign("POST", "/api/v1/sys/user/avatar"),
		apiSign("POST", "/api/v1/sys/user/switchAuthority"),
		apiSign("POST", "/api/v1/sys/user/addUser"),
		apiSign("PUT", "/api/v1/sys/user/updateUser"),
		apiSign("DELETE", "/api/v1/sys/user/deleteUser"),
		apiSign("POST", "/api/v1/sys/user/resetPassword"),
		apiSign("GET", "/api/v1/sys/menu/getMenu"),
		apiSign("POST", "/api/v1/sys/menu/getMenuList"),
		apiSign("POST", "/api/v1/sys/menu/getMenuAuthority"),
		apiSign("POST", "/api/v1/sys/menu/addBaseMenu"),
		apiSign("PUT", "/api/v1/sys/menu/updateBaseMenu"),
		apiSign("DELETE", "/api/v1/sys/menu/deleteBaseMenu"),
		apiSign("POST", "/api/v1/sys/authority/getAuthorityList"),
		apiSign("POST", "/api/v1/sys/authority/createAuthority"),
		apiSign("PUT", "/api/v1/sys/authority/updateAuthority"),
		apiSign("DELETE", "/api/v1/sys/authority/deleteAuthority"),
		apiSign("POST", "/api/v1/sys/authority/setAuthorityMenus"),
		apiSign("POST", "/api/v1/sys/api/getApiList"),
		apiSign("POST", "/api/v1/sys/api/createApi"),
		apiSign("PUT", "/api/v1/sys/api/updateApi"),
		apiSign("DELETE", "/api/v1/sys/api/deleteApi"),
		apiSign("POST", "/api/v1/sys/api-token/getApiTokenList"),
		apiSign("GET", "/api/v1/sys/api-token/detail"),
		apiSign("POST", "/api/v1/sys/api-token/create"),
		apiSign("PUT", "/api/v1/sys/api-token/update"),
		apiSign("DELETE", "/api/v1/sys/api-token/delete"),
		apiSign("POST", "/api/v1/sys/api-token/reset"),
		apiSign("POST", "/api/v1/sys/api-token/enable"),
		apiSign("POST", "/api/v1/sys/api-token/disable"),
		apiSign("POST", "/api/v1/sys/casbin/getPolicyPathByAuthorityId"),
		apiSign("POST", "/api/v1/sys/casbin/updateCasbin"),
		apiSign("POST", "/api/v1/sys/operationLog/getOperationLogList"),
		apiSign("DELETE", "/api/v1/sys/operationLog/deleteOperationLogByIds"),
		apiSign("POST", "/api/v1/sys/file/upload"),
		apiSign("POST", "/api/v1/sys/system/getServerInfo"),
		apiSign("POST", "/api/v1/sys/notice/createNotice"),
		apiSign("POST", "/api/v1/sys/notice/getNoticeList"),
		apiSign("GET", "/api/v1/sys/notice/getMyNotices"),
		apiSign("POST", "/api/v1/sys/notice/markRead"),
	}
	basicAccess := []string{
		apiSign("GET", "/api/v1/sys/user/getSelfInfo"),
		apiSign("POST", "/api/v1/sys/user/logout"),
		apiSign("PUT", "/api/v1/sys/user/info"),
		apiSign("PUT", "/api/v1/sys/user/ui-config"),
		apiSign("POST", "/api/v1/sys/user/avatar"),
		apiSign("GET", "/api/v1/sys/menu/getMenu"),
		apiSign("POST", "/api/v1/sys/system/getServerInfo"),
		apiSign("GET", "/api/v1/sys/notice/getMyNotices"),
		apiSign("POST", "/api/v1/sys/notice/markRead"),
	}

	roleAccess := map[uint][]string{
		adminAuthorityID: fullAccess,
		9528:             fullAccess,
		888:              basicAccess,
		8881:             basicAccess,
	}

	relations := make([]model.SysAuthorityApi, 0)
	for authorityID, signs := range roleAccess {
		for _, sign := range signs {
			apiID, ok := apiIDs[sign]
			if !ok {
				continue
			}
			relations = append(relations, model.SysAuthorityApi{AuthorityId: authorityID, ApiId: apiID})
		}
	}
	if len(relations) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&relations).Error
}

func ensureCasbinPolicies(tx *gorm.DB, apiIDs map[string]uint, adminAuthorityID uint) error {
	_ = apiIDs
	fullAccess := [][]string{
		{"GET", "/api/v1/sys/user/getSelfInfo"},
		{"POST", "/api/v1/sys/user/getUserList"},
		{"POST", "/api/v1/sys/user/logout"},
		{"PUT", "/api/v1/sys/user/info"},
		{"PUT", "/api/v1/sys/user/ui-config"},
		{"POST", "/api/v1/sys/user/avatar"},
		{"POST", "/api/v1/sys/user/switchAuthority"},
		{"POST", "/api/v1/sys/user/addUser"},
		{"PUT", "/api/v1/sys/user/updateUser"},
		{"DELETE", "/api/v1/sys/user/deleteUser"},
		{"POST", "/api/v1/sys/user/resetPassword"},
		{"GET", "/api/v1/sys/menu/getMenu"},
		{"POST", "/api/v1/sys/menu/getMenuList"},
		{"POST", "/api/v1/sys/menu/getMenuAuthority"},
		{"POST", "/api/v1/sys/menu/addBaseMenu"},
		{"PUT", "/api/v1/sys/menu/updateBaseMenu"},
		{"DELETE", "/api/v1/sys/menu/deleteBaseMenu"},
		{"POST", "/api/v1/sys/authority/getAuthorityList"},
		{"POST", "/api/v1/sys/authority/createAuthority"},
		{"PUT", "/api/v1/sys/authority/updateAuthority"},
		{"DELETE", "/api/v1/sys/authority/deleteAuthority"},
		{"POST", "/api/v1/sys/authority/setAuthorityMenus"},
		{"POST", "/api/v1/sys/api/getApiList"},
		{"POST", "/api/v1/sys/api/createApi"},
		{"PUT", "/api/v1/sys/api/updateApi"},
		{"DELETE", "/api/v1/sys/api/deleteApi"},
		{"POST", "/api/v1/sys/api-token/getApiTokenList"},
		{"GET", "/api/v1/sys/api-token/detail"},
		{"POST", "/api/v1/sys/api-token/create"},
		{"PUT", "/api/v1/sys/api-token/update"},
		{"DELETE", "/api/v1/sys/api-token/delete"},
		{"POST", "/api/v1/sys/api-token/reset"},
		{"POST", "/api/v1/sys/api-token/enable"},
		{"POST", "/api/v1/sys/api-token/disable"},
		{"POST", "/api/v1/sys/casbin/getPolicyPathByAuthorityId"},
		{"POST", "/api/v1/sys/casbin/updateCasbin"},
		{"POST", "/api/v1/sys/operationLog/getOperationLogList"},
		{"DELETE", "/api/v1/sys/operationLog/deleteOperationLogByIds"},
		{"POST", "/api/v1/sys/file/upload"},
		{"POST", "/api/v1/sys/system/getServerInfo"},
		{"POST", "/api/v1/sys/notice/createNotice"},
		{"POST", "/api/v1/sys/notice/getNoticeList"},
		{"GET", "/api/v1/sys/notice/getMyNotices"},
		{"POST", "/api/v1/sys/notice/markRead"},
	}
	basicAccess := [][]string{
		{"GET", "/api/v1/sys/user/getSelfInfo"},
		{"POST", "/api/v1/sys/user/logout"},
		{"PUT", "/api/v1/sys/user/info"},
		{"PUT", "/api/v1/sys/user/ui-config"},
		{"POST", "/api/v1/sys/user/avatar"},
		{"GET", "/api/v1/sys/menu/getMenu"},
		{"POST", "/api/v1/sys/system/getServerInfo"},
		{"GET", "/api/v1/sys/notice/getMyNotices"},
		{"POST", "/api/v1/sys/notice/markRead"},
	}

	rolePolicies := map[uint][][]string{
		adminAuthorityID: fullAccess,
		9528:             fullAccess,
		888:              basicAccess,
		8881:             basicAccess,
	}

	rules := make([]casbinRuleSeed, 0)
	for authorityID, policies := range rolePolicies {
		sub := strconv.FormatUint(uint64(authorityID), 10)
		for _, p := range policies {
			rules = append(rules, casbinRuleSeed{Ptype: "p", V0: sub, V1: p[1], V2: p[0], V3: "", V4: "", V5: ""})
		}
	}
	if len(rules) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rules).Error
}

func ensureAdminUser(tx *gorm.DB, opts seedAdminOptions) error {
	// A fresh CMS seeds only this administrator. Test users belong in isolated
	// test databases; rerunning initialization must preserve existing passwords.
	var existing model.SysUser
	err := tx.Where("username = ?", opts.Username).First(&existing).Error
	if err == nil {
		if existing.AuthorityID == 0 {
			if err := tx.Model(&existing).Update("authority_id", opts.AuthorityID).Error; err != nil {
				return err
			}
		}
		userAuth := model.SysUserAuthority{UserId: existing.ID, AuthorityId: opts.AuthorityID}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&userAuth).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	hashedPwd, err := utils.BcryptHash(opts.Password)
	if err != nil {
		return err
	}

	admin := model.SysUser{
		UUID:        uuid.New(),
		Username:    opts.Username,
		Password:    hashedPwd,
		NickName:    "Administrator",
		Avatar:      model.DefaultUserAvatar,
		Status:      model.UserActive,
		AuthorityID: opts.AuthorityID,
	}
	if err := tx.Create(&admin).Error; err != nil {
		return err
	}

	userAuth := model.SysUserAuthority{UserId: admin.ID, AuthorityId: opts.AuthorityID}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&userAuth).Error
}

func apiSign(method, path string) string {
	return strings.ToUpper(method) + " " + path
}
