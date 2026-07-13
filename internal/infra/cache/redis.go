// Package cache creates Redis clients used by infrastructure adapters.
package cache

import (
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRedis(addr, password string, db int) (*redis.Client, error) {
	if strings.TrimSpace(addr) == "" {
		return nil, fmt.Errorf("redis address is required")
	}
	if db < 0 {
		return nil, fmt.Errorf("redis database index must not be negative")
	}
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
	rdb.AddHook(newTracingHook(addr, db))

	return rdb, nil
}
