package middleware

import (
	"runtime/debug"

	"github.com/freeDog-wy/go-backend-template/internal/handler"
	"github.com/freeDog-wy/go-backend-template/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

// Recovery converts unexpected handler panics into a stable API response.
func Recovery(log logger.Logger) gin.HandlerFunc {
	if log == nil {
		log = logger.Noop()
	}

	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Error(
					"http request panicked",
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"client_ip", c.ClientIP(),
					"trace_id", traceID(c),
					"panic", recovered,
					"stack", string(debug.Stack()),
				)

				if c.Writer.Written() {
					c.Abort()
					return
				}
				handler.Fail(c, "INTERNAL_ERROR", "服务器内部错误")
			}
		}()

		c.Next()
	}
}

func traceID(c *gin.Context) string {
	spanContext := trace.SpanContextFromContext(c.Request.Context())
	if !spanContext.IsValid() {
		return ""
	}
	return spanContext.TraceID().String()
}
