// Package cache 提供 Redis 连接初始化。
// 业务接口由各使用方自行定义（captcha.Store / session.Store 等），
// 本包只负责创建 *redis.Client 并配置连接池。
package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedis 创建 Redis 客户端，自动检测连接是否可达。
func NewRedis(addr, password string, db int) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		PoolSize:     10,
		MinIdleConns: 3,
		MaxIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// 启动时做一次 ping，确保连接可达
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return rdb, nil
}
