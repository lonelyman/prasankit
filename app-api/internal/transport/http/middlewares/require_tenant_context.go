package middlewares

import (
	"strings"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/workspace"
	authhandler "prasankit-api/internal/transport/http/auth"
	"prasankit-api/internal/transport/http/presenter"
	workspacehandler "prasankit-api/internal/transport/http/workspace"

	"github.com/gofiber/fiber/v3"
)

// RequireTenantContext middleware runs AFTER RequireSession.
// It reads X-Workspace-Slug, resolves the workspace + membership from DB (D16 — fresh
// per request), and stores *workspace.TenantContext in c.Locals(LocalsKeyTenant).
//
// Error mapping:
//   - Missing/empty header      → 400 tenant.workspace_required
//   - Workspace not found       → 404 tenant.workspace_not_found
//   - No active membership      → 403 tenant.forbidden
func RequireTenantContext(
	wsRepo workspace.WorkspaceRepository,
	memberRepo workspace.MembershipRepository,
) fiber.Handler {
	return func(c fiber.Ctx) error {
		account, ok := c.Locals(authhandler.LocalsKeyAccount).(*auth.Account)
		if !ok || account == nil {
			return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
		}

		slug := strings.TrimSpace(c.Get("X-Workspace-Slug"))
		if slug == "" {
			return presenter.RenderError(c, fiber.StatusBadRequest, "tenant.workspace_required", "X-Workspace-Slug header is required")
		}

		ws, err := wsRepo.FindBySlugActive(c.Context(), slug)
		if err != nil {
			return err // unexpected — let 500 handler catch
		}
		if ws == nil {
			// Slug not found — 404 (slugs are public identifiers; distinguish from 403).
			return presenter.RenderError(c, fiber.StatusNotFound, "tenant.workspace_not_found", "Workspace not found")
		}

		membership, err := memberRepo.FindActiveByWorkspaceAndAccount(c.Context(), ws.ID, account.ID)
		if err != nil {
			return err
		}
		if membership == nil {
			// Workspace exists but account has no active membership → 403.
			return presenter.RenderError(c, fiber.StatusForbidden, "tenant.forbidden", "You are not a member of this workspace")
		}

		tc := &workspace.TenantContext{
			WorkspaceID:  ws.ID,
			MembershipID: membership.ID,
			AccountID:    account.ID,
			OrgRoleCode:  membership.OrgRoleCode,
		}
		c.Locals(workspacehandler.LocalsKeyTenant, tc)
		return c.Next()
	}
}
