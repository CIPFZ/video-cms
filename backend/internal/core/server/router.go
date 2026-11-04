package server

import (
	"fmt"
	"github.com/CIPFZ/video-cms/internal/modules/media"
	"net/http"
	"strings"

	tokenCore "github.com/CIPFZ/video-cms/internal/core/token"
	"github.com/CIPFZ/video-cms/internal/docs"
	"github.com/CIPFZ/video-cms/internal/middleware"
	systemApi "github.com/CIPFZ/video-cms/internal/modules/system/api"
	systemRepo "github.com/CIPFZ/video-cms/internal/modules/system/repository"
	systemRouter "github.com/CIPFZ/video-cms/internal/modules/system/router"
	systemService "github.com/CIPFZ/video-cms/internal/modules/system/service"
	"github.com/CIPFZ/video-cms/internal/svc"
	"github.com/CIPFZ/video-cms/pkg/response"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
)

func InitRouters(svcCtx *svc.ServiceContext) *gin.Engine {
	r := gin.New()
	if err := r.SetTrustedProxies(svcCtx.Config.System.TrustedProxies); err != nil {
		panic(err)
	}
	r.Use(gin.Recovery())

	registerHealthRoutes(r, svcCtx)
	registerGlobalMiddleware(r, svcCtx)
	registerBaseRoutes(r, svcCtx)

	routerPrefix := svcCtx.Config.System.RouterPrefix
	publicGroup := r.Group(routerPrefix)
	publicGroup.GET("/public/config", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"registrationEnabled": svcCtx.Config.System.AllowRegistration}})
	})
	publicGroup.Use(middleware.LoginRateLimit(svcCtx.Redis))
	privateGroup := r.Group(routerPrefix)
	privateGroup.Use(middleware.JWTAuth(svcCtx), middleware.CasbinHandler(svcCtx))

	// External identities never enter the CMS JWT/Casbin group.
	r.GET(tokenCore.TokenInfoPath(routerPrefix), middleware.ApiTokenAuth(svcCtx), func(c *gin.Context) {
		response.OkWithData(gin.H{"tokenId": c.GetUint(middleware.CtxKeyAPITokenID)}, c)
	})

	sysRouter := wireSystemModule(svcCtx)
	sysRouter.InitSystemRoutes(privateGroup, publicGroup)

	if svcCtx.MediaCollector != nil {
		mediaAPI := media.NewAPI(svcCtx.MediaCollector)
		privateGroup.GET("/media/sources", mediaAPI.ListSources)
		mediaWrite := privateGroup.Group("/media/sources", middleware.OperationRecord(svcCtx))
		mediaWrite.POST("", mediaAPI.CreateSource)
		mediaWrite.PUT("/:id", mediaAPI.UpdateSource)
		mediaWrite.POST("/:id/health", mediaAPI.Health)
		mediaWrite.POST("/:id/collect", mediaAPI.Enqueue)
		privateGroup.GET("/media/runs", mediaAPI.ListRuns)
		privateGroup.GET("/media/runs/:id/items", mediaAPI.ListRunItems)
		privateGroup.POST("/media/runs/:id/cancel", mediaAPI.CancelRun)
		privateGroup.GET("/media/schedules", mediaAPI.ListSchedules)
		privateGroup.PUT("/media/schedules/:id", mediaAPI.UpdateSchedule)
		privateGroup.GET("/media/sources/:id/health-history", mediaAPI.ListHealth)
		privateGroup.GET("/media/items", mediaAPI.ListItems)
		privateGroup.GET("/media/items/:id", mediaAPI.GetItem)
		privateGroup.POST("/media/items/:id/refresh", mediaAPI.RefreshItem)
		privateGroup.POST("/media/episodes/:id/check", mediaAPI.CheckEpisode)
		privateGroup.GET("/media/episodes/:id/stream", mediaAPI.StreamEpisode)
	}

	svcCtx.Routers = r.Routes()
	svcCtx.Logger.Info("all routes initialized")
	return r
}

func registerGlobalMiddleware(r *gin.Engine, svcCtx *svc.ServiceContext) {
	cfg := svcCtx.Config

	if cfg.Observable.Exporter != "none" {
		r.Use(otelgin.Middleware(
			cfg.Observable.ServiceName,
			otelgin.WithTracerProvider(otel.GetTracerProvider()),
			otelgin.WithMeterProvider(otel.GetMeterProvider()),
			otelgin.WithFilter(func(r *http.Request) bool {
				path := r.URL.Path
				return path != "/health" && path != "/metrics" && !strings.HasPrefix(path, "/swagger")
			}),
		))
	}

	r.Use(middleware.I18nMiddleware(svcCtx.I18n))
	r.Use(middleware.RateLimitMiddleware(cfg.RateLimit))
	r.Use(middleware.GinLoggerMiddleware(svcCtx.Logger))
	r.Use(middleware.CorsByRules(svcCtx.Config.Cors))
}

func registerBaseRoutes(r *gin.Engine, svcCtx *svc.ServiceContext) {
	routerPrefix := svcCtx.Config.System.RouterPrefix
	docs.SwaggerInfo.BasePath = routerPrefix

	r.GET(routerPrefix+"/swagger/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, routerPrefix+"/swagger/index.html")
	})
	r.GET(routerPrefix+"/swagger/index.html", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, swaggerIndexHTML(routerPrefix+"/swagger/doc.json"))
	})
	r.GET(routerPrefix+"/swagger/doc.json", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(docs.SwaggerInfo.ReadDoc()))
	})

}

func swaggerIndexHTML(docURL string) string {
	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: "%s",
      dom_id: "#swagger-ui",
      presets: [SwaggerUIBundle.presets.apis],
      layout: "BaseLayout"
    });
  </script>
</body>
</html>`, docURL)
}

func wireSystemModule(svcCtx *svc.ServiceContext) *systemRouter.SystemRouter {
	userRepo := systemRepo.NewUserRepository(svcCtx.DB)
	menuRepo := systemRepo.NewMenuRepository(svcCtx.DB)
	authRepo := systemRepo.NewAuthorityRepository(svcCtx.DB)
	apiRepo := systemRepo.NewApiRepository(svcCtx.DB)
	apiTokenRepo := systemRepo.NewApiTokenRepository(svcCtx.DB, svcCtx.Config.System.RouterPrefix)
	casbinRepo := systemRepo.NewCasbinRepository(svcCtx.DB)
	opLogRepo := systemRepo.NewOperationLogRepository(svcCtx.DB)
	noticeRepo := systemRepo.NewNoticeRepository(svcCtx.DB)

	opLogService := systemService.NewOperationLogService(opLogRepo)
	userService := systemService.NewUserService(svcCtx, userRepo)
	menuService := systemService.NewMenuService(svcCtx, menuRepo)
	authService := systemService.NewAuthorityService(authRepo)
	apiService := systemService.NewApiService(apiRepo)
	apiTokenService := systemService.NewApiTokenService(apiTokenRepo)
	casbinService := systemService.NewCasbinService(casbinRepo)
	noticeService := systemService.NewNoticeService(noticeRepo)

	apis := &systemRouter.SystemApis{
		UserApi:      systemApi.NewUserApi(svcCtx, userService),
		MenuApi:      systemApi.NewMenuApi(svcCtx, menuService),
		AuthorityApi: systemApi.NewAuthorityApi(svcCtx, authService),
		SysApiApi:    systemApi.NewSysApiApi(svcCtx, apiService),
		ApiTokenApi:  systemApi.NewApiTokenApi(svcCtx, apiTokenService),
		CasbinApi:    systemApi.NewCasbinApi(svcCtx, casbinService),
		OpLogApi:     systemApi.NewOperationLogApi(svcCtx, opLogService),
		FileApi:      systemApi.NewFileApi(svcCtx),
		StateApi:     systemApi.NewStateApi(svcCtx),
		NoticeApi:    systemApi.NewNoticeApi(svcCtx, noticeService),
	}

	return systemRouter.NewSystemRouter(svcCtx, apis)
}
