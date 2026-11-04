package db

import (
	"context"
	"fmt"
	"time"

	"github.com/CIPFZ/video-cms/internal/core/config"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

func InitRedis(redisCfg config.Redis) (redis.UniversalClient, error) {
	var instance redis.UniversalClient
	var err error
	// 使用集群模式
	if redisCfg.UseCluster {
		instance = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:                 redisCfg.ClusterAddrs,
			Username:              redisCfg.Username,
			Password:              redisCfg.Password,
			Protocol:              2,
			ContextTimeoutEnabled: true, DialTimeout: 3 * time.Second, ReadTimeout: 3 * time.Second, WriteTimeout: 3 * time.Second, MaxRetries: 1,
		})
	} else {
		// 使用单例模式
		instance = redis.NewClient(&redis.Options{
			Addr:                  redisCfg.Addr,
			Username:              redisCfg.Username,
			Password:              redisCfg.Password,
			DB:                    redisCfg.DB,
			Protocol:              2,
			ContextTimeoutEnabled: true, DialTimeout: 3 * time.Second, ReadTimeout: 3 * time.Second, WriteTimeout: 3 * time.Second, MaxRetries: 1,
		})
	}
	// 为客户端添加 Otel 钩子
	if err = redisotel.InstrumentTracing(instance); err != nil {
		_ = instance.Close()
		return nil, fmt.Errorf("failed to instrument redis with otel: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = instance.Ping(ctx).Result()
	if err != nil {
		_ = instance.Close()
		return nil, err
	}
	return instance, err
}
