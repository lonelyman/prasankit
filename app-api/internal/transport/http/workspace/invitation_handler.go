package workspacehandler

import (
	"errors"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/workspace"
	authhandler "prasankit-api/internal/transport/http/auth"
	"prasankit-api/internal/transport/http/presenter"

	"github.com/gofiber/fiber/v3"
)

// ── Request / response types ──────────────────────────────────────────────────

type inviteRequest struct {
	Email       string `json:"email"`
	OrgRoleCode string `json:"org_role_code"`
}

type invitationResponse struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	Email       string `json:"email"`
	OrgRoleCode string `json:"org_role_code"`
	ExpiresAt   string `json:"expires_at"`
}

type acceptInviteRequest struct {
	Token string `json:"token"`
}

type acceptInviteResponse struct {
	WorkspaceID string `json:"workspace_id"`
	OrgRoleCode string `json:"org_role_code"`
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// HandleInvite handles POST /workspaces/invitations
// Chain: requireSession → requireTenantContext → requireWorkspacePermission(invite).
func (h *Handler) HandleInvite(c fiber.Ctx) error {
	tc, ok := c.Locals(LocalsKeyTenant).(*workspace.TenantContext)
	if !ok || tc == nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "tenant.workspace_required", "Workspace context not resolved")
	}

	var req inviteRequest
	if err := c.Bind().JSON(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid request body")
	}

	result, err := h.svc.InviteMember(c.Context(), workspace.InviteMemberInput{
		TenantCtx:   *tc,
		Email:       req.Email,
		OrgRoleCode: req.OrgRoleCode,
		IP:          c.IP(),
		UserAgent:   string(c.Request().Header.UserAgent()),
		RequestID:   c.Get("X-Request-Id"),
	})
	if err != nil {
		return h.handleInviteError(c, err)
	}

	inv := result.Invitation
	return presenter.RenderItem(c, invitationResponse{
		ID:          inv.ID.String(),
		WorkspaceID: inv.WorkspaceID.String(),
		Email:       inv.Email,
		OrgRoleCode: inv.OrgRoleCode,
		ExpiresAt:   inv.ExpiresAt.Format("2006-01-02T15:04:05Z"),
	}, fiber.StatusCreated)
}

// HandleAcceptInvite handles POST /invitations/accept
// Chain: requireSession only (accepter is not yet a member).
func (h *Handler) HandleAcceptInvite(c fiber.Ctx) error {
	account, ok := c.Locals(authhandler.LocalsKeyAccount).(*auth.Account)
	if !ok || account == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}

	var req acceptInviteRequest
	if err := c.Bind().JSON(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid request body")
	}
	if req.Token == "" {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "token is required")
	}

	result, err := h.svc.AcceptInvitation(
		c.Context(),
		account.ID,
		account.PrimaryEmail,
		req.Token,
		c.IP(),
		string(c.Request().Header.UserAgent()),
		c.Get("X-Request-Id"),
	)
	if err != nil {
		return h.handleInviteError(c, err)
	}

	return presenter.RenderItem(c, acceptInviteResponse{
		WorkspaceID: result.WorkspaceID.String(),
		OrgRoleCode: result.OrgRoleCode,
	})
}

// handleInviteError maps invitation service errors to HTTP responses.
func (h *Handler) handleInviteError(c fiber.Ctx, err error) error {
	var valErr *workspace.ValidationError
	if errors.As(err, &valErr) {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Validation failed", valErr.Fields)
	}
	if errors.Is(err, workspace.ErrInviteRoleNotAllowed) {
		return presenter.RenderError(c, fiber.StatusBadRequest, "invitation.role_not_allowed", "The specified role is not allowed for invitations")
	}
	if errors.Is(err, workspace.ErrAlreadyMember) {
		return presenter.RenderError(c, fiber.StatusConflict, "invitation.already_member", "This account is already an active member of the workspace")
	}
	if errors.Is(err, workspace.ErrAlreadyPending) {
		return presenter.RenderError(c, fiber.StatusConflict, "invitation.already_pending", "A pending invitation already exists for this email")
	}
	if errors.Is(err, workspace.ErrInvitationInvalid) {
		return presenter.RenderError(c, fiber.StatusNotFound, "invitation.invalid", "Invitation not found or invalid")
	}
	if errors.Is(err, workspace.ErrInvitationExpired) {
		return presenter.RenderError(c, fiber.StatusGone, "invitation.expired", "Invitation has expired or is no longer active")
	}
	if errors.Is(err, workspace.ErrEmailMismatch) {
		return presenter.RenderError(c, fiber.StatusForbidden, "invitation.email_mismatch", "Your account email does not match the invitation")
	}
	return err
}
