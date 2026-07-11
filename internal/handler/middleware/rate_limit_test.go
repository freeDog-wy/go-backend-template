package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Run("rejects over limit before calling the handler", func(t *testing.T) {
		limiter := &stubRateLimiter{allowed: false}
		handlerCalled := false
		router := gin.New()
		router.Use(RateLimit(limiter, nil, true, 2, time.Minute, []RateLimitPolicy{{Method: http.MethodPost, Path: "/login", Scope: "login"}}))
		router.POST("/login", func(c *gin.Context) { handlerCalled = true; c.Status(http.StatusNoContent) })

		response := serveRequest(router, http.MethodPost, "/login")
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"code":"RATE_LIMITED"`) {
			t.Fatalf("response = %d %s", response.Code, response.Body.String())
		}
		if handlerCalled {
			t.Fatal("handler was called after rate limit rejection")
		}
		if limiter.scope != "login" || limiter.subject != "192.0.2.1" {
			t.Fatalf("limiter inputs = (%q, %q)", limiter.scope, limiter.subject)
		}
	})

	t.Run("fails closed when the limiter is unavailable", func(t *testing.T) {
		limiter := &stubRateLimiter{err: errors.New("redis unavailable")}
		router := gin.New()
		router.Use(RateLimit(limiter, nil, true, 2, time.Minute, []RateLimitPolicy{{Method: http.MethodPost, Path: "/login", Scope: "login"}}))
		router.POST("/login", func(c *gin.Context) { c.Status(http.StatusNoContent) })

		response := serveRequest(router, http.MethodPost, "/login")
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"code":"RATE_LIMIT_UNAVAILABLE"`) {
			t.Fatalf("response = %d %s", response.Code, response.Body.String())
		}
	})

	t.Run("does not apply to routes without a policy", func(t *testing.T) {
		limiter := &stubRateLimiter{allowed: false}
		router := gin.New()
		router.Use(RateLimit(limiter, nil, true, 2, time.Minute, nil))
		router.GET("/open", func(c *gin.Context) { c.Status(http.StatusNoContent) })

		response := serveRequest(router, http.MethodGet, "/open")
		if response.Code != http.StatusNoContent || limiter.calls != 0 {
			t.Fatalf("response = %d, limiter calls = %d", response.Code, limiter.calls)
		}
	})
}

type stubRateLimiter struct {
	allowed bool
	err     error
	calls   int
	scope   string
	subject string
}

func (l *stubRateLimiter) Allow(_ context.Context, scope, subject string, _ int, _ time.Duration) (bool, error) {
	l.calls++
	l.scope = scope
	l.subject = subject
	return l.allowed, l.err
}

func serveRequest(router *gin.Engine, method, path string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, nil)
	request.RemoteAddr = "192.0.2.1:12345"
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}
