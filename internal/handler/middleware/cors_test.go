package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

const allowedTestOrigin = "https://admin.example.test"

func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("handles an allowed preflight without running the handler", func(t *testing.T) {
		handlerCalled := false
		router := corsRouter(&handlerCalled)
		request := httptest.NewRequest(http.MethodOptions, "/admin", nil)
		request.Header.Set("Origin", allowedTestOrigin)
		request.Header.Set("Access-Control-Request-Method", http.MethodPatch)
		request.Header.Set("Access-Control-Request-Headers", "authorization, content-type, idempotency-key")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusNoContent || handlerCalled {
			t.Fatalf("response = %d, handler called = %t", response.Code, handlerCalled)
		}
		if response.Header().Get("Access-Control-Allow-Origin") != allowedTestOrigin || response.Header().Get("Access-Control-Allow-Credentials") != "true" {
			t.Fatalf("CORS headers = %#v", response.Header())
		}
		vary := strings.Join(response.Header().Values("Vary"), ",")
		for _, value := range []string{"Origin", "Access-Control-Request-Method", "Access-Control-Request-Headers"} {
			if !strings.Contains(vary, value) {
				t.Fatalf("Vary = %q, missing %q", vary, value)
			}
		}
	})

	t.Run("rejects an untrusted origin before the handler", func(t *testing.T) {
		handlerCalled := false
		router := corsRouter(&handlerCalled)
		request := httptest.NewRequest(http.MethodPost, "/admin", nil)
		request.Header.Set("Origin", "https://admin.example.test.attacker.invalid")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusForbidden || handlerCalled || response.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatalf("response = %d, handler called = %t, headers = %#v", response.Code, handlerCalled, response.Header())
		}
	})

	t.Run("rejects a preflight that asks for an unapproved header", func(t *testing.T) {
		handlerCalled := false
		router := corsRouter(&handlerCalled)
		request := httptest.NewRequest(http.MethodOptions, "/admin", nil)
		request.Header.Set("Origin", allowedTestOrigin)
		request.Header.Set("Access-Control-Request-Method", http.MethodPost)
		request.Header.Set("Access-Control-Request-Headers", "X-Admin-Secret")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusForbidden || handlerCalled {
			t.Fatalf("response = %d, handler called = %t", response.Code, handlerCalled)
		}
	})

	t.Run("allows a configured origin to reach the handler", func(t *testing.T) {
		handlerCalled := false
		router := corsRouter(&handlerCalled)
		request := httptest.NewRequest(http.MethodGet, "/admin", nil)
		request.Header.Set("Origin", allowedTestOrigin)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusNoContent || !handlerCalled || response.Header().Get("Access-Control-Allow-Origin") != allowedTestOrigin {
			t.Fatalf("response = %d, handler called = %t, headers = %#v", response.Code, handlerCalled, response.Header())
		}
	})

	t.Run("does not affect a request without an Origin header", func(t *testing.T) {
		handlerCalled := false
		router := corsRouter(&handlerCalled)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/admin", nil))

		if response.Code != http.StatusNoContent || !handlerCalled || response.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatalf("response = %d, handler called = %t, headers = %#v", response.Code, handlerCalled, response.Header())
		}
	})
}

func corsRouter(handlerCalled *bool) *gin.Engine {
	router := gin.New()
	router.Use(CORS([]string{allowedTestOrigin}))
	router.GET("/admin", func(c *gin.Context) { *handlerCalled = true; c.Status(http.StatusNoContent) })
	router.POST("/admin", func(c *gin.Context) { *handlerCalled = true; c.Status(http.StatusNoContent) })
	router.PATCH("/admin", func(c *gin.Context) { *handlerCalled = true; c.Status(http.StatusNoContent) })
	return router
}
