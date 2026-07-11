package admin_role

import (
	"context"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
	svcAuth "github.com/freeDog-wy/go-backend-template/internal/usecase/auth"
	svcAuthorization "github.com/freeDog-wy/go-backend-template/internal/usecase/authorization"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpdateRoleRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := New(&roleAuth{}, &roleService{}, &roleService{})
	h.RegisterRoutes(r)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/roles/not-a-number", strings.NewReader(`{"name":"Role"}`))
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"code":"INVALID_INPUT"`) {
		t.Fatalf("response = %d %s", w.Code, w.Body.String())
	}
}

type roleAuth struct{}

func (*roleAuth) AuthenticateAccessToken(context.Context, string) (*svcAuth.AccessIdentity, error) {
	return &svcAuth.AccessIdentity{UserID: 1}, nil
}

type roleService struct{}

func (*roleService) EnsureAdminAccess(context.Context, uint) error             { return nil }
func (*roleService) HasPermission(context.Context, uint, string) (bool, error) { return true, nil }
func (*roleService) ListRoles(context.Context, svcAuthorization.ListRolesCmd) ([]*svcAuthorization.RoleResult, shared.PageResult, error) {
	return nil, shared.PageResult{}, nil
}
func (*roleService) CreateRole(context.Context, svcAuthorization.CreateRoleCmd) (*svcAuthorization.RoleResult, error) {
	return nil, nil
}
func (*roleService) UpdateRole(context.Context, svcAuthorization.UpdateRoleCmd) (*svcAuthorization.RoleResult, error) {
	return nil, nil
}
func (*roleService) ListPermissions(context.Context, svcAuthorization.ListPermissionsCmd) ([]*svcAuthorization.PermissionResult, shared.PageResult, error) {
	return nil, shared.PageResult{}, nil
}
