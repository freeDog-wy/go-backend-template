//go:build integration

package ratelimit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/freeDog-wy/go-backend-template/internal/testsupport"
)

func TestRateLimiterIntegration(t *testing.T) {
	ctx := context.Background()
	rdb := testsupport.OpenRedis(t)
	prefix := fmt.Sprintf("integration:rate-limit:%d", time.Now().UnixNano())
	limiter := NewRateLimiter(rdb, prefix)
	keys := []string{
		limiter.key("login", "192.0.2.1"),
		limiter.key("register", "192.0.2.1"),
	}
	t.Cleanup(func() { _ = rdb.Del(ctx, keys...).Err() })

	for attempt := 1; attempt <= 2; attempt++ {
		allowed, err := limiter.Allow(ctx, "login", "192.0.2.1", 2, time.Minute)
		if err != nil || !allowed {
			t.Fatalf("login attempt %d = (%t, %v), want (true, nil)", attempt, allowed, err)
		}
	}
	allowed, err := limiter.Allow(ctx, "login", "192.0.2.1", 2, time.Minute)
	if err != nil || allowed {
		t.Fatalf("third login attempt = (%t, %v), want (false, nil)", allowed, err)
	}

	allowed, err = limiter.Allow(ctx, "register", "192.0.2.1", 2, time.Minute)
	if err != nil || !allowed {
		t.Fatalf("register scope = (%t, %v), want (true, nil)", allowed, err)
	}

	ttl, err := rdb.PTTL(ctx, limiter.key("login", "192.0.2.1")).Result()
	if err != nil || ttl <= 0 {
		t.Fatalf("login rate limit TTL = (%v, %v), want a positive TTL", ttl, err)
	}
}
