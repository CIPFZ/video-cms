package svc

import (
	"github.com/CIPFZ/video-cms/internal/core/file"
	"github.com/CIPFZ/video-cms/internal/modules/media"
	"net/http"
	"sync"
	"time"

	"github.com/CIPFZ/video-cms/internal/core/audit"
	"github.com/CIPFZ/video-cms/internal/core/config"
	"github.com/CIPFZ/video-cms/internal/core/i18n"
	"github.com/CIPFZ/video-cms/internal/core/jwt"
	coretoken "github.com/CIPFZ/video-cms/internal/core/token"

	"github.com/CIPFZ/video-cms/internal/core/claims"
	"github.com/CIPFZ/video-cms/internal/core/session"
	"github.com/gin-gonic/gin"
	"github.com/qiniu/qmgo"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

type ServiceContext struct {
	MediaCollector     *media.Service
	SRV                *http.Server
	Config             *config.Config
	Viper              *viper.Viper
	JWT                *jwt.JWT
	Logger             *zap.Logger
	I18n               *i18n.Service
	DB                 *gorm.DB
	Redis              redis.UniversalClient
	Mongo              *qmgo.QmgoClient
	Routers            gin.RoutesInfo
	Timer              time.Timer
	ConcurrencyControl *singleflight.Group
	Policy             *claims.PolicyManager
	Sessions           *session.Store
	APITokenLimiter    *coretoken.InMemoryLimiter
	lock               sync.RWMutex
	AuditRecorder      *audit.AuditRecorder
	OSS                file.OSS
}

func NewServiceContext() *ServiceContext {
	return &ServiceContext{
		ConcurrencyControl: &singleflight.Group{},
		APITokenLimiter:    coretoken.NewInMemoryLimiter(),
	}
}
