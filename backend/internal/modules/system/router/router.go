package router

import (
	"github.com/CIPFZ/video-cms/internal/middleware"
	"github.com/CIPFZ/video-cms/internal/modules/system/api"
	"github.com/CIPFZ/video-cms/internal/svc"
	"github.com/gin-gonic/gin"
)

// SystemApis 聚合了 system 模块所需的所有 API 处理器
type SystemApis struct {
	UserApi      *api.UserApi
	MenuApi      *api.MenuApi
	AuthorityApi *api.AuthorityApi
	SysApiApi    *api.SysApiApi
	ApiTokenApi  *api.ApiTokenApi
	CasbinApi    *api.CasbinApi
	OpLogApi     *api.OperationLogApi
	FileApi      *api.FileApi
	StateApi     *api.StateApi
	NoticeApi    *api.NoticeApi
}

// SystemRouter 负责注册 system 模块的所有路由
type SystemRouter struct {
	svcCtx *svc.ServiceContext
	apis   *SystemApis
}

// NewSystemRouter 构造函数，接收一个包含所有 API 处理器的聚合结构体
func NewSystemRouter(svcCtx *svc.ServiceContext, apis *SystemApis) *SystemRouter {
	return &SystemRouter{
		svcCtx: svcCtx,
		apis:   apis,
	}
}

// InitSystemRoutes 初始化 system 模块的路由
// 它将路由分为公开路由（如登录、注册）和私有路由（需要认证）
func (s *SystemRouter) InitSystemRoutes(privateGroup *gin.RouterGroup, publicGroup *gin.RouterGroup) {
	// --- 注册公开路由 (Public Routes) ---
	// 这些路由不需要 JWT 认证
	s.initPublicRoutes(publicGroup)

	// --- 注册私有路由 (Private Routes) ---
	// 所有这些路由都经过 JWTAuthMiddleware 和 CasbinMiddleware 的保护
	s.initPrivateRoutes(privateGroup)
}

// initPublicRoutes 注册 system 模块的公开访问路由
func (s *SystemRouter) initPublicRoutes(public *gin.RouterGroup) {
	// 创建 /user 路由组
	userRouter := public.Group("user")
	{
		// @Tags User
		// @Summary 用户登录
		// @Router /user/login [post]
		userRouter.POST("login", s.apis.UserApi.Login)

		// @Tags User
		// @Summary 用户注册
		// @Router /user/register [post]
		userRouter.POST("register", s.apis.UserApi.Register)
	}
}

// initPrivateRoutes 注册 system 模块的需要认证的路由
func (s *SystemRouter) initPrivateRoutes(private *gin.RouterGroup) {
	// 创建一个总的 /system 路由组，使模块路由更加内聚
	systemGroup := private.Group("sys")

	// 在 systemGroup 下，为不同的资源创建子路由组
	s.initUserRoutes(systemGroup)
	s.initMenuRoutes(systemGroup)
	s.initAuthorityRoutes(systemGroup)
	s.initApiRoutes(systemGroup)
	s.initApiTokenRoutes(systemGroup)
	s.initCasbinRoutes(systemGroup)
	s.initOperationLogRoutes(systemGroup)
	s.initFileUploadRoutes(systemGroup)
	s.initStateRoutes(systemGroup)
	s.initNoticeRoutes(systemGroup)
}

// initUserRoutes 注册用户管理相关路由
func (s *SystemRouter) initUserRoutes(group *gin.RouterGroup) {
	userRouter := group.Group("user")
	{
		// --- "读" 操作 ---
		userRouter.GET("getSelfInfo", s.apis.UserApi.GetSelfInfo)
		userRouter.POST("getUserList", s.apis.UserApi.GetUserList)
		userRouter.POST("logout", s.apis.UserApi.Logout)
		userRouter.PUT("info", s.apis.UserApi.UpdateSelfInfo)
		userRouter.PUT("ui-config", s.apis.UserApi.UpdateUiConfig)
		userRouter.POST("avatar", s.apis.UserApi.UploadAvatar)

		// --- "写" 操作 (统一应用操作日志中间件) ---
		userWriteGroup := userRouter.Group("", middleware.OperationRecord(s.svcCtx))
		{
			userWriteGroup.POST("switchAuthority", s.apis.UserApi.SwitchAuthority)
			userWriteGroup.POST("addUser", s.apis.UserApi.AddUser)
			userWriteGroup.PUT("updateUser", s.apis.UserApi.UpdateUser)    // 建议: PUT
			userWriteGroup.DELETE("deleteUser", s.apis.UserApi.DeleteUser) // 建议: DELETE
			userWriteGroup.POST("resetPassword", s.apis.UserApi.ResetPassword)
		}
	}
}

// initMenuRoutes 注册菜单管理相关路由
func (s *SystemRouter) initMenuRoutes(group *gin.RouterGroup) {
	menuRouter := group.Group("menu")
	{
		// --- "读" 操作 ---
		menuRouter.GET("getMenu", s.apis.MenuApi.GetMenu)                    // 获取当前用户菜单
		menuRouter.POST("getMenuList", s.apis.MenuApi.GetMenuList)           // 获取所有菜单, 建议: GET
		menuRouter.POST("getMenuAuthority", s.apis.MenuApi.GetMenuAuthority) // 获取指定角色菜单, 建议: GET

		// --- "写" 操作 (统一应用操作日志中间件) ---
		menuWriteGroup := menuRouter.Group("", middleware.OperationRecord(s.svcCtx))
		{
			menuWriteGroup.POST("addBaseMenu", s.apis.MenuApi.AddBaseMenu)
			menuWriteGroup.PUT("updateBaseMenu", s.apis.MenuApi.UpdateBaseMenu)    // 建议: PUT
			menuWriteGroup.DELETE("deleteBaseMenu", s.apis.MenuApi.DeleteBaseMenu) // 建议: DELETE
		}
	}
}

// initAuthorityRoutes 注册角色管理相关路由
func (s *SystemRouter) initAuthorityRoutes(group *gin.RouterGroup) {
	authRouter := group.Group("authority")
	{
		// --- "读" 操作 ---
		authRouter.POST("getAuthorityList", s.apis.AuthorityApi.GetAuthorityList) // 建议: GET

		// --- "写" 操作 (统一应用操作日志中间件) ---
		authWriteGroup := authRouter.Group("", middleware.OperationRecord(s.svcCtx))
		{
			authWriteGroup.POST("createAuthority", s.apis.AuthorityApi.CreateAuthority)
			authWriteGroup.PUT("updateAuthority", s.apis.AuthorityApi.UpdateAuthority)      // 建议: PUT
			authWriteGroup.DELETE("deleteAuthority", s.apis.AuthorityApi.DeleteAuthority)   // 建议: DELETE
			authWriteGroup.POST("setAuthorityMenus", s.apis.AuthorityApi.SetAuthorityMenus) // 分配菜单权限
		}
	}
}

// initApiRoutes 注册 API 管理相关路由
func (s *SystemRouter) initApiRoutes(group *gin.RouterGroup) {
	apiRouter := group.Group("api")
	{
		// --- "读" 操作 ---
		apiRouter.POST("getApiList", s.apis.SysApiApi.GetApiList) // 建议: GET

		// --- "写" 操作 (统一应用操作日志中间件) ---
		apiWriteGroup := apiRouter.Group("", middleware.OperationRecord(s.svcCtx))
		{
			apiWriteGroup.POST("createApi", s.apis.SysApiApi.CreateApi)
			apiWriteGroup.PUT("updateApi", s.apis.SysApiApi.UpdateApi)    // 建议: PUT
			apiWriteGroup.DELETE("deleteApi", s.apis.SysApiApi.DeleteApi) // 建议: DELETE
		}
	}
}

func (s *SystemRouter) initApiTokenRoutes(group *gin.RouterGroup) {
	apiTokenRouter := group.Group("api-token")
	{
		apiTokenRouter.POST("getApiTokenList", s.apis.ApiTokenApi.GetApiTokenList)
		apiTokenRouter.POST("options", s.apis.ApiTokenApi.GetOptions)
		apiTokenRouter.GET("detail", s.apis.ApiTokenApi.GetApiTokenDetail)

		apiTokenWriteGroup := apiTokenRouter.Group("", middleware.OperationRecord(s.svcCtx))
		{
			apiTokenWriteGroup.POST("create", s.apis.ApiTokenApi.CreateApiToken)
			apiTokenWriteGroup.PUT("update", s.apis.ApiTokenApi.UpdateApiToken)
			apiTokenWriteGroup.DELETE("delete", s.apis.ApiTokenApi.DeleteApiToken)
			apiTokenWriteGroup.POST("reset", s.apis.ApiTokenApi.ResetApiToken)
			apiTokenWriteGroup.POST("enable", s.apis.ApiTokenApi.EnableApiToken)
			apiTokenWriteGroup.POST("disable", s.apis.ApiTokenApi.DisableApiToken)
		}
	}
}

// initCasbinRoutes 注册 Casbin 策略管理相关路由
func (s *SystemRouter) initCasbinRoutes(group *gin.RouterGroup) {
	casbinRouter := group.Group("casbin")
	{
		// --- "读" 操作 ---
		casbinRouter.POST("getPolicyPathByAuthorityId", s.apis.CasbinApi.GetPolicyPathByAuthorityId) // 建议: GET

		// --- "写" 操作 (统一应用操作日志中间件) ---
		casbinWriteGroup := casbinRouter.Group("", middleware.OperationRecord(s.svcCtx))
		{
			casbinWriteGroup.POST("updateCasbin", s.apis.CasbinApi.UpdateCasbin) // 分配 API 权限
		}
	}
}

// initOperationLogRoutes 注册操作日志管理相关路由
func (s *SystemRouter) initOperationLogRoutes(group *gin.RouterGroup) {
	opLogRouter := group.Group("operationLog")
	{
		// --- "读" 操作 ---
		opLogRouter.POST("getOperationLogList", s.apis.OpLogApi.GetOperationLogList) // 建议: GET

		// --- "写" 操作 (统一应用操作日志中间件) ---
		opLogWriteGroup := opLogRouter.Group("", middleware.OperationRecord(s.svcCtx))
		{
			opLogWriteGroup.DELETE("deleteOperationLogByIds", s.apis.OpLogApi.DeleteOperationLogByIds) // 建议: DELETE
		}
	}
}

// initFileUploadRoutes 注册文件上传相关路由
func (s *SystemRouter) initFileUploadRoutes(group *gin.RouterGroup) {
	group.POST("file/upload", s.apis.FileApi.Upload)
}

func (s *SystemRouter) initStateRoutes(group *gin.RouterGroup) {
	stateRouter := group.Group("system")
	{
		stateRouter.POST("getServerInfo", s.apis.StateApi.GetServerInfo)
	}
}

func (s *SystemRouter) initNoticeRoutes(group *gin.RouterGroup) {
	noticeRouter := group.Group("notice")
	{
		noticeRouter.GET("getMyNotices", s.apis.NoticeApi.GetMyNotices)
		noticeRouter.POST("getNoticeList", s.apis.NoticeApi.GetNoticeList)
		noticeRouter.POST("markRead", s.apis.NoticeApi.MarkRead)

		noticeWriteGroup := noticeRouter.Group("", middleware.OperationRecord(s.svcCtx))
		{
			noticeWriteGroup.POST("createNotice", s.apis.NoticeApi.CreateNotice)
		}
	}
}
