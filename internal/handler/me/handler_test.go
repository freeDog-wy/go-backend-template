package me

import (
	"context"
	svcAuth "github.com/freeDog-wy/go-backend-template/internal/usecase/auth"
	svcIdentity "github.com/freeDog-wy/go-backend-template/internal/usecase/identity"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetProfileRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Run("requires token", func(t *testing.T) {
		r := gin.New()
		New(&meAuth{}, &meAuth{}, &meIdentity{}).RegisterRoutes(r)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/me", nil))
		if !strings.Contains(w.Body.String(), `"code":"UNAUTHORIZED"`) {
			t.Fatalf("response = %s", w.Body.String())
		}
	})
	t.Run("returns profile", func(t *testing.T) {
		r := gin.New()
		New(&meAuth{}, &meAuth{}, &meIdentity{user: &svcIdentity.UserResult{ID: 2, Name: "Alice", Email: "a@example.com"}}).RegisterRoutes(r)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		req.Header.Set("Authorization", "Bearer token")
		r.ServeHTTP(w, req)
		if !strings.Contains(w.Body.String(), `"email":"a@example.com"`) {
			t.Fatalf("response = %s", w.Body.String())
		}
	})
}

type meAuth struct{}

func (*meAuth) AuthenticateAccessToken(context.Context, string) (*svcAuth.AccessIdentity, error) {
	return &svcAuth.AccessIdentity{UserID: 2}, nil
}
func (*meAuth) ChangePassword(context.Context, svcAuth.ChangePasswordCmd) error { return nil }

type meIdentity struct{ user *svcIdentity.UserResult }

func (f *meIdentity) GetByID(context.Context, uint) (*svcIdentity.UserResult, error) {
	return f.user, nil
}
func (*meIdentity) UpdateProfile(context.Context, svcIdentity.UpdateProfileCmd) (*svcIdentity.UserResult, error) {
	return nil, nil
}
