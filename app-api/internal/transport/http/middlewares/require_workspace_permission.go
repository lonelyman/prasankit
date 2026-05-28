package middlewares

import (
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/internal/transport/http/presenter"
	workspacehandler "prasankit-api/internal/transport/http/workspace"

	"github.com/gofiber/fiber/v3"
)

// RequireWorkspacePermission returns a middleware that verifies the authenticated member
// (resolved by RequireTenantContext) holds the given permission.
//
// Chain: RequireSession → RequireTenantContext → RequireWorkspacePermission(perm) → handler.
//
// Error mapping:
//   - TenantContext missing (misconfigured chain) → 401 auth.unauthenticated
//   - Role not in allowed set for perm            → 403 tenant.permission_denied
func RequireWorkspacePermission(perm string) fiber.Handler {
	return func(c fiber.Ctx) error {
		tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
		if !ok || tc == nil {
			return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
		}

		if !workspace.IsRoleAllowedFor(perm, tc.OrgRoleCode) {
			return presenter.RenderError(c, fiber.StatusForbidden, "tenant.permission_denied", "You do not have permission to perform this action")
		}

		return c.Next()
	}
}
