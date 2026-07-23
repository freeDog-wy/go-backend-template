package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	svcAuth "github.com/freeDog-wy/go-backend-template/internal/usecase/auth"
	svcIdentity "github.com/freeDog-wy/go-backend-template/internal/usecase/identity"
	svcVerification "github.com/freeDog-wy/go-backend-template/internal/usecase/verification"
	"github.com/gin-gonic/gin"
)

func TestLoginRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Run("returns auth response", func(t *testing.T) {
		r := gin.New()
		h := New(&authFake{result: &svcAuth.AuthResult{AccessToken: "access", RefreshToken: "refresh", User: &svcIdentity.UserResult{ID: 1, Email: "a@example.com"}}}, &authzFake{}, &identityFake{}, &verificationFake{})
		h.RegisterRoutes(r)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"a@example.com","password":"secret1"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"access_token":"access"`) {
			t.Fatalf("response = %d %s", w.Code, w.Body.String())
		}
	})
	t.Run("maps invalid credentials", func(t *testing.T) {
		r := gin.New()
		h := New(&authFake{err: svcAuth.ErrInvalidCredentials}, &authzFake{}, &identityFake{}, &verificationFake{})
		h.RegisterRoutes(r)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"a@example.com","password":"secret1"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if !strings.Contains(w.Body.String(), `"code":"INVALID_CREDENTIALS"`) {
			t.Fatalf("response = %s", w.Body.String())
		}
	})
}

func TestAdminCookieSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	newHandler := func(auth *authFake) *Handler {
		return NewWithCookieOptions(auth, &authzFake{}, &identityFake{}, &verificationFake{}, CookieOptions{
			AdminOrigin: "https://admin.example.test",
			Name:        "admin_refresh_token",
			Secure:      true,
			TTL:         7 * 24 * time.Hour,
		})
	}

	t.Run("admin login writes an HttpOnly refresh cookie", func(t *testing.T) {
		r := gin.New()
		h := newHandler(&authFake{result: authResult("access", "refresh")})
		h.RegisterRoutes(r)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/login", strings.NewReader(`{"email":"a@example.com","password":"secret1"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if !strings.Contains(w.Body.String(), `"access_token":"access"`) || strings.Contains(w.Body.String(), "refresh_token") {
			t.Fatalf("response = %s", w.Body.String())
		}
		cookies := w.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatalf("cookies = %d, want 1", len(cookies))
		}
		cookie := cookies[0]
		if cookie.Name != "admin_refresh_token" || cookie.Value != "refresh" || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != refreshCookiePath {
			t.Fatalf("cookie = %+v", cookie)
		}
	})

	t.Run("refreshes from an allowed cookie and rotates it", func(t *testing.T) {
		r := gin.New()
		auth := &authFake{refreshResult: authResult("next-access", "next-refresh")}
		h := newHandler(auth)
		h.RegisterRoutes(r)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
		req.Header.Set("Origin", "https://admin.example.test")
		req.AddCookie(&http.Cookie{Name: "admin_refresh_token", Value: "refresh"})
		r.ServeHTTP(w, req)

		if auth.refreshCommand.RefreshToken != "refresh" {
			t.Fatalf("refresh command = %+v", auth.refreshCommand)
		}
		if !strings.Contains(w.Body.String(), `"access_token":"next-access"`) || strings.Contains(w.Body.String(), "next-refresh") {
			t.Fatalf("response = %s", w.Body.String())
		}
		cookies := w.Result().Cookies()
		if len(cookies) != 1 || cookies[0].Value != "next-refresh" {
			t.Fatalf("cookies = %+v", cookies)
		}
	})

	t.Run("rejects a cookie request from an untrusted origin", func(t *testing.T) {
		r := gin.New()
		auth := &authFake{refreshResult: authResult("next-access", "next-refresh")}
		h := newHandler(auth)
		h.RegisterRoutes(r)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
		req.Header.Set("Origin", "https://attacker.example.test")
		req.AddCookie(&http.Cookie{Name: "admin_refresh_token", Value: "refresh"})
		r.ServeHTTP(w, req)

		if !strings.Contains(w.Body.String(), `"code":"FORBIDDEN"`) || auth.refreshCommand.RefreshToken != "" {
			t.Fatalf("response = %s, refresh command = %+v", w.Body.String(), auth.refreshCommand)
		}
	})

	t.Run("logs out from an allowed cookie and clears it", func(t *testing.T) {
		r := gin.New()
		auth := &authFake{}
		h := newHandler(auth)
		h.RegisterRoutes(r)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		req.Header.Set("Origin", "https://admin.example.test")
		req.AddCookie(&http.Cookie{Name: "admin_refresh_token", Value: "refresh"})
		r.ServeHTTP(w, req)

		if auth.logoutCommand.RefreshToken != "refresh" {
			t.Fatalf("logout command = %+v", auth.logoutCommand)
		}
		cookies := w.Result().Cookies()
		if len(cookies) != 1 || cookies[0].Name != "admin_refresh_token" || cookies[0].MaxAge >= 0 || cookies[0].Path != refreshCookiePath {
			t.Fatalf("cookies = %+v", cookies)
		}
	})

	t.Run("continues to accept the existing JSON refresh request", func(t *testing.T) {
		r := gin.New()
		auth := &authFake{refreshResult: authResult("next-access", "next-refresh")}
		h := newHandler(auth)
		h.RegisterRoutes(r)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(`{"refresh_token":"body-refresh"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if auth.refreshCommand.RefreshToken != "body-refresh" || !strings.Contains(w.Body.String(), `"refresh_token":"next-refresh"`) || len(w.Result().Cookies()) != 0 {
			t.Fatalf("response = %s, refresh command = %+v, cookies = %+v", w.Body.String(), auth.refreshCommand, w.Result().Cookies())
		}
	})
}

func authResult(accessToken, refreshToken string) *svcAuth.AuthResult {
	return &svcAuth.AuthResult{AccessToken: accessToken, RefreshToken: refreshToken, User: &svcIdentity.UserResult{ID: 1, Email: "a@example.com"}}
}

type authFake struct {
	result         *svcAuth.AuthResult
	err            error
	refreshResult  *svcAuth.AuthResult
	refreshErr     error
	refreshCommand svcAuth.RefreshCmd
	logoutCommand  svcAuth.LogoutCmd
}

func (f *authFake) Login(context.Context, svcAuth.LoginCmd) (*svcAuth.AuthResult, error) {
	return f.result, f.err
}
func (f *authFake) Refresh(_ context.Context, command svcAuth.RefreshCmd) (*svcAuth.AuthResult, error) {
	f.refreshCommand = command
	return f.refreshResult, f.refreshErr
}
func (f *authFake) Logout(_ context.Context, command svcAuth.LogoutCmd) error {
	f.logoutCommand = command
	return nil
}

type authzFake struct{ err error }

func (f *authzFake) EnsureAdminAccess(context.Context, uint) error           { return f.err }
func (*authzFake) HasPermission(context.Context, uint, string) (bool, error) { return false, nil }

type identityFake struct{}

func (*identityFake) Register(context.Context, svcIdentity.RegisterCmd) (*svcIdentity.UserResult, error) {
	return nil, errors.New("unused")
}

type verificationFake struct{}

func (*verificationFake) ResendVerification(context.Context, svcVerification.ResendVerificationCmd) error {
	return nil
}
func (*verificationFake) VerifyEmail(context.Context, svcVerification.VerifyEmailCmd) error {
	return nil
}
func (*verificationFake) ForgotPassword(context.Context, svcVerification.ForgotPasswordCmd) error {
	return nil
}
func (*verificationFake) ResetPassword(context.Context, svcVerification.ResetPasswordCmd) error {
	return nil
}
