package middleware

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/CIPFZ/video-cms/internal/core/config"
	"github.com/CIPFZ/video-cms/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
	"net/http"
	"sync"
	"time"
)

type ipEntry struct {
	limiter *rate.Limiter
	seen    time.Time
}
type IPRateLimiter struct {
	mu        sync.Mutex
	ips       map[string]ipEntry
	qps       rate.Limit
	burst     int
	nextSweep time.Time
	overflow  *rate.Limiter
}

func NewIPRateLimiter(qps, burst int) *IPRateLimiter { return newIPLimiter(rate.Limit(qps), burst) }
func newIPLimiter(qps rate.Limit, burst int) *IPRateLimiter {
	return &IPRateLimiter{ips: make(map[string]ipEntry), qps: qps, burst: burst, overflow: rate.NewLimiter(qps, burst)}
}
func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()
	now := time.Now()
	if now.After(i.nextSweep) {
		for key, e := range i.ips {
			if now.Sub(e.seen) > 10*time.Minute {
				delete(i.ips, key)
			}
		}
		i.nextSweep = now.Add(time.Minute)
	}
	if e, ok := i.ips[ip]; ok {
		e.seen = now
		i.ips[ip] = e
		return e.limiter
	}
	if len(i.ips) >= 10000 {
		return i.overflow
	}
	l := rate.NewLimiter(i.qps, i.burst)
	i.ips[ip] = ipEntry{l, now}
	return l
}
func rateDenied(c *gin.Context) {
	c.Header("Retry-After", "60")
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"code": 7, "msg": response.LocalizeMessage(c, "访问过于频繁，请稍后再试"), "data": nil})
}
func RateLimitMiddleware(cfg config.RateLimitConfig) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) { c.Next() }
	}
	limiter := NewIPRateLimiter(cfg.QPS, cfg.Burst)
	return func(c *gin.Context) {
		if !limiter.GetLimiter(c.ClientIP()).Allow() {
			rateDenied(c)
			return
		}
		c.Next()
	}
}

var loginQuota = redis.NewScript(`local n=redis.call('INCR',KEYS[1]);if n==1 then redis.call('EXPIRE',KEYS[1],60) end;return n`)

// Login/register use a separate small quota. Redis shares it across replicas;
// dependency failure closes authentication rather than bypassing throttling.
func LoginRateLimit(client redis.UniversalClient) gin.HandlerFunc {
	local := newIPLimiter(rate.Every(3*time.Second), 20)
	return func(c *gin.Context) {
		if client != nil {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()
			key := fmt.Sprintf("cms:login:%x", sha256.Sum256([]byte(c.ClientIP())))
			n, err := loginQuota.Run(ctx, client, []string{key}).Int()
			if err != nil {
				c.AbortWithStatus(http.StatusServiceUnavailable)
				return
			}
			if n > 20 {
				rateDenied(c)
				return
			}
		} else if !local.GetLimiter(c.ClientIP()).Allow() {
			rateDenied(c)
			return
		}
		c.Next()
	}
}
