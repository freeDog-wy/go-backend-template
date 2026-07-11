package testsupport

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/freeDog-wy/go-backend-template/pkg/envfile"
	"github.com/redis/go-redis/v9"
)

const (
	redisAddrEnv     = "TEST_REDIS_ADDR"
	redisPasswordEnv = "TEST_REDIS_PASSWORD"
	redisDBEnv       = "TEST_REDIS_DB"
)

// OpenRedis opens the Redis database explicitly configured for integration tests.
// Tests must clean only the keys they create and must never flush the database.
func OpenRedis(t testing.TB) *redis.Client {
	t.Helper()
	if err := envfile.LoadNearest(".env"); err != nil {
		t.Fatalf("load nearest .env: %v", err)
	}

	addr := strings.TrimSpace(os.Getenv(redisAddrEnv))
	if addr == "" {
		t.Fatalf("%s must be set for Redis integration tests", redisAddrEnv)
	}
	dbValue := strings.TrimSpace(os.Getenv(redisDBEnv))
	if dbValue == "" {
		t.Fatalf("%s must be set for Redis integration tests", redisDBEnv)
	}
	db, err := strconv.Atoi(dbValue)
	if err != nil || db < 0 {
		t.Fatalf("%s must be a non-negative integer", redisDBEnv)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv(redisPasswordEnv),
		DB:       db,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		t.Fatalf("ping Redis at %s db %d: %v", addr, db, err)
	}

	t.Cleanup(func() { _ = client.Close() })
	return client
}
