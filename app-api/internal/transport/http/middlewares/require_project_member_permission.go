package middlewares

import (
	"prasankit-api/internal/modules/projectmember"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/internal/transport/http/presenter"
	workspacehandler "prasankit-api/internal/transport/http/workspace"

	"github.com/gofiber/fiber/v3"
)

// RequireProjectMemberPermission gates a route on whether the caller's org_role in
// the resolved TenantContext is allowed for the given project_member permission code.
// Sister to RequireProjectPermission but consults projectmember.IsRoleAllowedFor
// instead. Distinct middleware (one per module) is the deliberate pattern —
// each module owns its own permission map, which keeps the authz surface
// per-module-reviewable.
func RequireProjectMemberPermission(perm string) fiber.Handler {
	return func(c fiber.Ctx) error {
		tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
		if !ok || tc == nil {
			return presenter.RenderError(c, fiber.StatusUnauthorized,
				"auth.unauthenticated", "Not authenticated")
		}
		if !projectmember.IsRoleAllowedFor(perm, tc.OrgRoleCode) {
			return presenter.RenderError(c, fiber.StatusForbidden,
				"tenant.permission_denied", "You do not have permission to perform this action")
		}
		return c.Next()
	}
}
