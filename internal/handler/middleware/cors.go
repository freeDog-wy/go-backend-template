package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var corsAllowedMethods = []string{
	http.MethodGet,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodOptions,
}

var corsAllowedHeaders = []string{
	"Content-Type",
	"Authorization",
	IdempotencyKeyHeader,
	"X-Correlation-ID",
}

// CORS applies a strict browser-origin allowlist. Requests without an Origin
// header are not browser cross-origin requests and retain their normal behavior.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	origins := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origins[origin] = struct{}{}
	}
	allowedMethods := make(map[string]struct{}, len(corsAllowedMethods))
	for _, method := range corsAllowedMethods {
		allowedMethods[method] = struct{}{}
	}
	allowedHeaders := make(map[string]struct{}, len(corsAllowedHeaders))
	for _, header := range corsAllowedHeaders {
		allowedHeaders[strings.ToLower(header)] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}
		if _, allowed := origins[origin]; !allowed {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		setCORSResponseHeaders(c, origin)
		if c.Request.Method != http.MethodOptions {
			c.Next()
			return
		}
		if !isAllowedPreflight(c, allowedMethods, allowedHeaders) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Header("Access-Control-Max-Age", strconv.Itoa(int((10 * time.Minute).Seconds())))
		c.Status(http.StatusNoContent)
		c.Abort()
	}
}

func setCORSResponseHeaders(c *gin.Context, origin string) {
	c.Header("Access-Control-Allow-Origin", origin)
	c.Header("Access-Control-Allow-Credentials", "true")
	c.Header("Access-Control-Allow-Methods", strings.Join(corsAllowedMethods, ", "))
	c.Header("Access-Control-Allow-Headers", strings.Join(corsAllowedHeaders, ", "))
	c.Writer.Header().Add("Vary", "Origin")
}

func isAllowedPreflight(c *gin.Context, methods, headers map[string]struct{}) bool {
	method := strings.ToUpper(strings.TrimSpace(c.GetHeader("Access-Control-Request-Method")))
	if _, allowed := methods[method]; !allowed {
		return false
	}
	for _, header := range strings.Split(c.GetHeader("Access-Control-Request-Headers"), ",") {
		header = strings.ToLower(strings.TrimSpace(header))
		if header == "" {
			continue
		}
		if _, allowed := headers[header]; !allowed {
			return false
		}
	}
	c.Writer.Header().Add("Vary", "Access-Control-Request-Method")
	c.Writer.Header().Add("Vary", "Access-Control-Request-Headers")
	return true
}
