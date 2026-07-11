package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/freeDog-wy/go-backend-template/pkg/logger"
	"github.com/gin-gonic/gin"
)

func TestRecoveryConvertsPanicToInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := &recordingLogger{}
	router := gin.New()
	router.Use(Recovery(log))
	router.GET("/panic", func(*gin.Context) { panic("unexpected failure") })

	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if body := response.Body.String(); !strings.Contains(body, `"code":"INTERNAL_ERROR"`) || strings.Contains(body, "unexpected failure") {
		t.Fatalf("response body = %q", body)
	}
	if len(log.errors) != 1 {
		t.Fatalf("error logs = %d, want 1", len(log.errors))
	}
	if !hasLogValue(log.errors[0], "panic", "unexpected failure") {
		t.Fatalf("panic value was not logged: %#v", log.errors[0])
	}
}

type recordingLogger struct {
	errors [][]any
}

func (l *recordingLogger) Info(string, ...any)  {}
func (l *recordingLogger) Debug(string, ...any) {}
func (l *recordingLogger) Error(_ string, args ...any) {
	l.errors = append(l.errors, append([]any(nil), args...))
}
func (l *recordingLogger) With(...any) logger.Logger { return l }

func hasLogValue(args []any, key string, want any) bool {
	for i := 0; i+1 < len(args); i += 2 {
		if args[i] == key && args[i+1] == want {
			return true
		}
	}
	return false
}
