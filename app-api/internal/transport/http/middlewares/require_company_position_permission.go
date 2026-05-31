package middlewares

import (
	"prasankit-api/internal/modules/companyposition"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/internal/transport/http/presenter"
	workspacehandler "prasankit-api/internal/transport/http/workspace"

	"github.com/gofiber/fiber/v3"
)

// RequireCompanyPositionPermission gates a route on whether the caller's org_role in the
// resolved TenantContext is allowed for the given company_position permission code. Also gates
// the ws company_position attach route (PermissionAssignCompanyPosition).
func RequireCompanyPositionPermission(perm string) fiber.Handler {
	return func(c fiber.Ctx) error {
		tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
		if !ok || tc == nil {
			return presenter.RenderError(c, fiber.StatusUnauthorized,
				"auth.unauthenticated", "Not authenticated")
		}
		if !companyposition.IsRoleAllowedFor(perm, tc.OrgRoleCode) {
			return presenter.RenderError(c, fiber.StatusForbidden,
				"tenant.permission_denied", "You do not have permission to perform this action")
		}
		return c.Next()
	}
}
