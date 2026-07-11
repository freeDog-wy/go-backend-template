package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

type authFake struct {
	result *svcAuth.AuthResult
	err    error
}

func (f *authFake) Login(context.Context, svcAuth.LoginCmd) (*svcAuth.AuthResult, error) {
	return f.result, f.err
}
func (f *authFake) Refresh(context.Context, svcAuth.RefreshCmd) (*svcAuth.AuthResult, error) {
	return nil, nil
}
func (f *authFake) Logout(context.Context, svcAuth.LogoutCmd) error { return nil }

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
