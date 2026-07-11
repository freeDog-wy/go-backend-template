package health

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
)

const defaultCheckTimeout = 2 * time.Second

type Checker interface {
	Check(context.Context) error
}

type CheckFunc func(context.Context) error

func (f CheckFunc) Check(ctx context.Context) error {
	return f(ctx)
}

type Handler struct {
	checks  map[string]Checker
	timeout time.Duration
}

func New(checks map[string]Checker, timeout time.Duration) *Handler {
	if timeout <= 0 {
		timeout = defaultCheckTimeout
	}
	cloned := make(map[string]Checker, len(checks))
	for name, check := range checks {
		if check != nil {
			cloned[name] = check
		}
	}
	return &Handler{checks: cloned, timeout: timeout}
}

func (h *Handler) RegisterRoutes(route *gin.Engine) {
	route.GET("/healthz", h.Live)
	route.GET("/readyz", h.Ready)
}

// Live reports whether the HTTP process can serve requests.
func (h *Handler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Ready reports whether the dependencies required by the HTTP server are reachable.
func (h *Handler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	type result struct {
		name string
		err  error
	}
	results := make(chan result, len(h.checks))
	for name, check := range h.checks {
		go func() {
			results <- result{name: name, err: check.Check(ctx)}
		}()
	}

	dependencies := make(map[string]string, len(h.checks))
	pending := sortedCheckNames(h.checks)
	for len(pending) > 0 {
		select {
		case result := <-results:
			if result.err != nil {
				dependencies[result.name] = "failed"
			} else {
				dependencies[result.name] = "ok"
			}
			pending = removePending(pending, result.name)
		case <-ctx.Done():
			for _, name := range pending {
				dependencies[name] = "failed"
			}
			pending = nil
		}
	}

	for _, status := range dependencies {
		if status != "ok" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "dependencies": dependencies})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready", "dependencies": dependencies})
}

func sortedCheckNames(checks map[string]Checker) []string {
	names := make([]string, 0, len(checks))
	for name := range checks {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func removePending(names []string, target string) []string {
	for index, name := range names {
		if name == target {
			return append(names[:index], names[index+1:]...)
		}
	}
	return names
}
