// Package ratelimit provides reusable request rate limiting primitives.
package ratelimit

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var incrementWithExpiry = redis.NewScript(`
local count = redis.call("INCR", KEYS[1])
if count == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
return count
`)

type RateLimiter struct {
	client *redis.Client
	prefix string
}

type Limiter interface {
	Allow(ctx context.Context, scope, subject string, limit int, window time.Duration) (bool, error)
}

func NewRateLimiter(client *redis.Client, prefix string) *RateLimiter {
	if strings.TrimSpace(prefix) == "" {
		prefix = "rate_limit"
	}
	return &RateLimiter{client: client, prefix: strings.Trim(prefix, ":")}
}

// Allow records one request in the subject's current window and returns whether it is within limit.
func (l *RateLimiter) Allow(ctx context.Context, scope, subject string, limit int, window time.Duration) (bool, error) {
	if l == nil || l.client == nil {
		return false, errors.New("rate limiter client is not configured")
	}
	if strings.TrimSpace(scope) == "" || strings.TrimSpace(subject) == "" {
		return false, errors.New("rate limit scope and subject must not be empty")
	}
	if limit <= 0 || window <= 0 {
		return false, errors.New("rate limit and window must be greater than zero")
	}

	count, err := incrementWithExpiry.Run(ctx, l.client, []string{l.key(scope, subject)}, window.Milliseconds()).Int64()
	if err != nil {
		return false, err
	}
	return count <= int64(limit), nil
}

func (l *RateLimiter) key(scope, subject string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(subject)))
	return fmt.Sprintf("%s:%s:%x", l.prefix, strings.TrimSpace(scope), sum[:])
}
