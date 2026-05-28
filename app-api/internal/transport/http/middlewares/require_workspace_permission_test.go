package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"prasankit-api/internal/modules/workspace"
	"prasankit-api/internal/transport/http/middlewares"
	workspacehandler "prasankit-api/internal/transport/http/workspace"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// buildPermMiddlewareApp wires a minimal fiber app: a single GET route protected by
// RequireTenantContext (simulated by pre-seeding Locals) + RequireWorkspacePermission.
func buildPermMiddlewareApp(perm string) *fiber.App {
	app := fiber.New()

	// Inject a TenantContext with the given role via a setup middleware.
	withRole := func(role string) fiber.Handler {
		return func(c fiber.Ctx) error {
			tc := &workspace.TenantContext{
				WorkspaceID: uuid.New(),
				AccountID:   uuid.New(),
				OrgRoleCode: role,
			}
			c.Locals(workspacehandler.LocalsKeyTenant, tc)
			return c.Next()
		}
	}

	requirePerm := middlewares.RequireWorkspacePermission(perm)
	okHandler := func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) }

	app.Get("/owner", withRole(workspace.OrgRoleOwner), requirePerm, okHandler)
	app.Get("/admin", withRole(workspace.OrgRoleAdmin), requirePerm, okHandler)
	app.Get("/user", withRole(workspace.OrgRoleUser), requirePerm, okHandler)
	app.Get("/executive", withRole(workspace.OrgRoleExecutive), requirePerm, okHandler)
	app.Get("/no-tenant", requirePerm, okHandler) // no TenantContext set
	return app
}

func TestRequireWorkspacePermission_InviteMember(t *testing.T) {
	app := buildPermMiddlewareApp(workspace.PermissionInviteMember)

	cases := []struct {
		path       string
		wantStatus int
		wantCode   string
	}{
		{"/owner", http.StatusOK, ""},
		{"/admin", http.StatusOK, ""},
		{"/user", http.StatusForbidden, "tenant.permission_denied"},
		{"/executive", http.StatusForbidden, "tenant.permission_denied"},
		{"/no-tenant", http.StatusUnauthorized, "auth.unauthenticated"},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("%s: app.Test: %v", tc.path, err)
		}
		resp.Body.Close()

		if resp.StatusCode != tc.wantStatus {
			t.Errorf("%s: status = %d, want %d", tc.path, resp.StatusCode, tc.wantStatus)
		}
	}
}
