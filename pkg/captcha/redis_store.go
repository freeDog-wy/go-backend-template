package captcha

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore 基于 Redis 的验证码存储，实现 base64Captcha.Store 接口。
type RedisStore struct {
	rdb    *redis.Client
	prefix string
	ttl    time.Duration
}

// NewRedisStore 创建 Redis 验证码存储。
// prefix 示例："captcha:"，最终 key 为 "captcha:<id>"。
func NewRedisStore(rdb *redis.Client, prefix string, ttl time.Duration) *RedisStore {
	return &RedisStore{rdb: rdb, prefix: prefix, ttl: ttl}
}

func (s *RedisStore) key(id string) string {
	return s.prefix + id
}

func (s *RedisStore) Set(id string, value string) error {
	return s.rdb.Set(context.Background(), s.key(id), value, s.ttl).Err()
}

func (s *RedisStore) Get(id string, clear bool) string {
	ctx := context.Background()
	key := s.key(id)
	val, err := s.rdb.Get(ctx, key).Result()
	if err != nil {
		return ""
	}
	if clear {
		s.rdb.Del(ctx, key)
	}
	return val
}

func (s *RedisStore) Verify(id, answer string, clear bool) bool {
	stored := s.Get(id, clear)
	if stored == "" {
		return false
	}
	return strings.EqualFold(stored, answer)
}
