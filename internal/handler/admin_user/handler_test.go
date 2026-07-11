package admin_user

import (
	"context"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
	svcAuth "github.com/freeDog-wy/go-backend-template/internal/usecase/auth"
	svcAuthorization "github.com/freeDog-wy/go-backend-template/internal/usecase/authorization"
	svcIdentity "github.com/freeDog-wy/go-backend-template/internal/usecase/identity"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateUserRequiresPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := New(&userAuth{}, &userAuthorization{allowed: false}, &userAuthorization{allowed: false}, &userIdentity{})
	h.RegisterRoutes(r)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", strings.NewReader(`{"name":"Alice","email":"a@example.com","password":"secret1"}`))
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), `"code":"FORBIDDEN"`) {
		t.Fatalf("response = %s", w.Body.String())
	}
}

type userAuth struct{}

func (*userAuth) AuthenticateAccessToken(context.Context, string) (*svcAuth.AccessIdentity, error) {
	return &svcAuth.AccessIdentity{UserID: 1}, nil
}

type userAuthorization struct{ allowed bool }

func (*userAuthorization) EnsureAdminAccess(context.Context, uint) error { return nil }
func (f *userAuthorization) HasPermission(context.Context, uint, string) (bool, error) {
	return f.allowed, nil
}
func (*userAuthorization) ReplaceUserRoles(context.Context, svcAuthorization.ReplaceUserRolesCmd) error {
	return nil
}
func (*userAuthorization) ListUserRoles(context.Context, uint) ([]*svcAuthorization.RoleResult, error) {
	return nil, nil
}

type userIdentity struct{}

func (*userIdentity) List(context.Context, svcIdentity.ListUsersCmd) ([]*svcIdentity.UserResult, shared.PageResult, error) {
	return nil, shared.PageResult{}, nil
}
func (*userIdentity) CreateAdminUser(context.Context, svcIdentity.CreateAdminUserCmd) (*svcIdentity.UserResult, error) {
	return nil, nil
}
func (*userIdentity) UpdateStatus(context.Context, svcIdentity.UpdateStatusCmd) (*svcIdentity.UserResult, error) {
	return nil, nil
}
