package health

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

func TestHealthRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Run("liveness does not check dependencies", func(t *testing.T) {
		router := newRouter(map[string]Checker{"database": CheckFunc(func(context.Context) error { return errors.New("unavailable") })})
		response := request(router, "/healthz")
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"status":"ok"`) {
			t.Fatalf("response = %d %s", response.Code, response.Body.String())
		}
	})

	t.Run("ready when all dependencies are available", func(t *testing.T) {
		router := newRouter(map[string]Checker{
			"database": CheckFunc(func(context.Context) error { return nil }),
			"redis":    CheckFunc(func(context.Context) error { return nil }),
		})
		response := request(router, "/readyz")
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"status":"ready"`) {
			t.Fatalf("response = %d %s", response.Code, response.Body.String())
		}
	})

	t.Run("returns service unavailable when a dependency fails", func(t *testing.T) {
		router := newRouter(map[string]Checker{
			"database": CheckFunc(func(context.Context) error { return errors.New("database unavailable") }),
			"redis":    CheckFunc(func(context.Context) error { return nil }),
		})
		response := request(router, "/readyz")
		if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), `"database":"failed"`) {
			t.Fatalf("response = %d %s", response.Code, response.Body.String())
		}
	})

	t.Run("returns service unavailable when checks exceed the timeout", func(t *testing.T) {
		router := gin.New()
		New(map[string]Checker{
			"database": CheckFunc(func(context.Context) error {
				time.Sleep(100 * time.Millisecond)
				return nil
			}),
		}, 10*time.Millisecond).RegisterRoutes(router)
		response := request(router, "/readyz")
		if response.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
		}
	})
}

func newRouter(checks map[string]Checker) *gin.Engine {
	router := gin.New()
	New(checks, time.Second).RegisterRoutes(router)
	return router
}

func request(router *gin.Engine, path string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	return response
}
