package middlewares

import (
	"prasankit-api/internal/modules/projectmemberposition"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/internal/transport/http/presenter"
	workspacehandler "prasankit-api/internal/transport/http/workspace"

	"github.com/gofiber/fiber/v3"
)

// RequireProjectMemberPositionPermission gates a route on whether the caller's org_role in the
// resolved TenantContext is allowed for the given project_member_position (junction) permission code.
func RequireProjectMemberPositionPermission(perm string) fiber.Handler {
	return func(c fiber.Ctx) error {
		tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
		if !ok || tc == nil {
			return presenter.RenderError(c, fiber.StatusUnauthorized,
				"auth.unauthenticated", "Not authenticated")
		}
		if !projectmemberposition.IsRoleAllowedFor(perm, tc.OrgRoleCode) {
			return presenter.RenderError(c, fiber.StatusForbidden,
				"tenant.permission_denied", "You do not have permission to perform this action")
		}
		return c.Next()
	}
}
