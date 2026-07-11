package middleware

import (
	"net/http"
	"time"

	"github.com/freeDog-wy/go-backend-template/internal/handler"
	"github.com/freeDog-wy/go-backend-template/pkg/logger"
	"github.com/freeDog-wy/go-backend-template/pkg/ratelimit"
	"github.com/gin-gonic/gin"
)

type RateLimitPolicy struct {
	Method string
	Path   string
	Scope  string
}

// RateLimit applies route-specific limits using the client IP as the subject.
func RateLimit(limiter ratelimit.Limiter, log logger.Logger, enabled bool, limit int, window time.Duration, policies []RateLimitPolicy) gin.HandlerFunc {
	if !enabled {
		return func(c *gin.Context) { c.Next() }
	}
	if log == nil {
		log = logger.Noop()
	}

	policyByRoute := make(map[string]RateLimitPolicy, len(policies))
	for _, policy := range policies {
		policyByRoute[policy.Method+" "+policy.Path] = policy
	}

	return func(c *gin.Context) {
		policy, ok := policyByRoute[c.Request.Method+" "+c.Request.URL.Path]
		if !ok {
			c.Next()
			return
		}

		allowed, err := limiter.Allow(c.Request.Context(), policy.Scope, c.ClientIP(), limit, window)
		if err != nil {
			log.Error("http rate limit check failed", "scope", policy.Scope, "client_ip", c.ClientIP(), "error", err)
			c.Abort()
			handler.Fail(c, "RATE_LIMIT_UNAVAILABLE", "请求暂时无法处理，请稍后重试")
			return
		}
		if !allowed {
			c.Abort()
			handler.Fail(c, "RATE_LIMITED", "请求过于频繁，请稍后重试")
			return
		}

		c.Next()
	}
}

var DefaultRateLimitPolicies = []RateLimitPolicy{
	{Method: http.MethodGet, Path: "/api/v1/captcha", Scope: "captcha"},
	{Method: http.MethodPost, Path: "/api/v1/auth/register", Scope: "register"},
	{Method: http.MethodPost, Path: "/api/v1/auth/resend-verification", Scope: "resend-verification"},
	{Method: http.MethodPost, Path: "/api/v1/auth/verify-email", Scope: "verify-email"},
	{Method: http.MethodPost, Path: "/api/v1/auth/forgot-password", Scope: "forgot-password"},
	{Method: http.MethodPost, Path: "/api/v1/auth/reset-password", Scope: "reset-password"},
	{Method: http.MethodPost, Path: "/api/v1/auth/login", Scope: "login"},
	{Method: http.MethodPost, Path: "/api/v1/admin/auth/login", Scope: "login"},
	{Method: http.MethodPost, Path: "/api/v1/auth/refresh", Scope: "refresh"},
}
