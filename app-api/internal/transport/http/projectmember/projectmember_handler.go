// Package projectmemberhandler exposes HTTP routes for the project_member module.
package projectmemberhandler

import (
	"errors"
	"time"

	"prasankit-api/internal/modules/projectmember"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/internal/transport/http/presenter"
	workspacehandler "prasankit-api/internal/transport/http/workspace"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// Handler handles /workspaces/projects/:id/members routes.
type Handler struct {
	svc *projectmember.Service
}

// NewHandler constructs a Handler.
func NewHandler(svc *projectmember.Service) *Handler {
	return &Handler{svc: svc}
}

// ── Request types ────────────────────────────────────────────────────────────

type addMemberRequest struct {
	WorkspaceMembershipID string `json:"workspace_membership_id"`
	ProjectRoleCode       string `json:"project_role_code"`
}

type changeRoleRequest struct {
	NewRoleCode string `json:"new_role_code"`
}

// ── Response types ────────────────────────────────────────────────────────────

type memberResponse struct {
	ID                    string  `json:"id"`
	WorkspaceID           string  `json:"workspace_id"`
	ProjectID             string  `json:"project_id"`
	WorkspaceMembershipID string  `json:"workspace_membership_id"`
	ProjectRoleCode       string  `json:"project_role_code"`
	JoinedAt              string  `json:"joined_at"`
	RemovedAt             *string `json:"removed_at"`
	DisplayName           *string `json:"display_name,omitempty"`
}

type listMembersResponse struct {
	Items []memberResponse `json:"items"`
	Count int              `json:"count"`
}

func toMemberResponse(m projectmember.ProjectMember) memberResponse {
	resp := memberResponse{
		ID:                    m.ID.String(),
		WorkspaceID:           m.WorkspaceID.String(),
		ProjectID:             m.ProjectID.String(),
		WorkspaceMembershipID: m.WorkspaceMembershipID.String(),
		ProjectRoleCode:       m.ProjectRoleCode,
		JoinedAt:              m.JoinedAt.UTC().Format(time.RFC3339),
	}
	if m.RemovedAt != nil {
		s := m.RemovedAt.UTC().Format(time.RFC3339)
		resp.RemovedAt = &s
	}
	return resp
}

func toMemberWithDisplayNameResponse(m projectmember.MemberWithDisplayName) memberResponse {
	resp := toMemberResponse(m.ProjectMember)
	dn := m.DisplayName
	resp.DisplayName = &dn
	return resp
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// HandleAdd handles POST /api/v1/workspaces/projects/:id/members.
func (h *Handler) HandleAdd(c fiber.Ctx) error {
	tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
	if !ok || tc == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}

	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid project id")
	}

	var req addMemberRequest
	if err := c.Bind().JSON(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid request body")
	}

	membershipID, err := uuid.Parse(req.WorkspaceMembershipID)
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid workspace_membership_id")
	}

	m, err := h.svc.AddMember(c.Context(), projectmember.AddMemberInput{
		TenantCtx:             *tc,
		ProjectID:             projectID,
		WorkspaceMembershipID: membershipID,
		ProjectRoleCode:       req.ProjectRoleCode,
		IP:                    c.IP(),
		UserAgent:             string(c.Request().Header.UserAgent()),
		RequestID:             c.Get("X-Request-Id"),
	})
	if err != nil {
		return h.handleServiceError(c, err)
	}
	return presenter.RenderItem(c, toMemberResponse(*m), fiber.StatusCreated)
}

// HandleList handles GET /api/v1/workspaces/projects/:id/members.
func (h *Handler) HandleList(c fiber.Ctx) error {
	tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
	if !ok || tc == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}

	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid project id")
	}

	members, err := h.svc.ListMembers(c.Context(), projectmember.ListMembersInput{
		TenantCtx: *tc,
		ProjectID: projectID,
	})
	if err != nil {
		return h.handleServiceError(c, err)
	}

	items := make([]memberResponse, 0, len(members))
	for _, m := range members {
		items = append(items, toMemberWithDisplayNameResponse(m))
	}
	return presenter.RenderItem(c, listMembersResponse{Items: items, Count: len(items)})
}

// HandleChangeRole handles PUT /api/v1/workspaces/projects/:id/members/:memberId/role.
func (h *Handler) HandleChangeRole(c fiber.Ctx) error {
	tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
	if !ok || tc == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}

	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid project id")
	}
	memberID, err := uuid.Parse(c.Params("memberId"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid member id")
	}

	var req changeRoleRequest
	if err := c.Bind().JSON(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid request body")
	}

	m, err := h.svc.ChangeRole(c.Context(), projectmember.ChangeRoleInput{
		TenantCtx:   *tc,
		ProjectID:   projectID,
		MemberID:    memberID,
		NewRoleCode: req.NewRoleCode,
		IP:          c.IP(),
		UserAgent:   string(c.Request().Header.UserAgent()),
		RequestID:   c.Get("X-Request-Id"),
	})
	if err != nil {
		return h.handleServiceError(c, err)
	}
	return presenter.RenderItem(c, toMemberResponse(*m))
}

// HandleRemove handles DELETE /api/v1/workspaces/projects/:id/members/:memberId.
// Returns 204 (empty body) on success.
func (h *Handler) HandleRemove(c fiber.Ctx) error {
	tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
	if !ok || tc == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}

	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid project id")
	}
	memberID, err := uuid.Parse(c.Params("memberId"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid member id")
	}

	err = h.svc.RemoveMember(c.Context(), projectmember.RemoveMemberInput{
		TenantCtx: *tc,
		ProjectID: projectID,
		MemberID:  memberID,
		IP:        c.IP(),
		UserAgent: string(c.Request().Header.UserAgent()),
		RequestID: c.Get("X-Request-Id"),
	})
	if err != nil {
		return h.handleServiceError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ── helpers ────────────────────────────────────────────────────────────────────

// handleServiceError maps projectmember service errors to HTTP responses.
func (h *Handler) handleServiceError(c fiber.Ctx, err error) error {
	var valErr *projectmember.ValidationError
	switch {
	case errors.As(err, &valErr):
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Validation failed",
			map[string]any{"fields": valErr.Fields})
	case errors.Is(err, projectmember.ErrMembershipNotEligible):
		return presenter.RenderError(c, fiber.StatusUnprocessableEntity, "project_member.membership_not_eligible", "Workspace membership is not eligible",
			map[string]any{"field": "workspace_membership_id"})
	case errors.Is(err, projectmember.ErrInvalidRoleCode):
		return presenter.RenderError(c, fiber.StatusUnprocessableEntity, "project_member.invalid_role_code", "Invalid project_role_code",
			map[string]any{"field": "project_role_code"})
	case errors.Is(err, projectmember.ErrOwnerRoleImmutable):
		return presenter.RenderError(c, fiber.StatusUnprocessableEntity, "project_member.owner_immutable", "Cannot modify the project owner outside ownership transfer")
	case errors.Is(err, projectmember.ErrAlreadyMember):
		return presenter.RenderError(c, fiber.StatusConflict, "project_member.already_member", "Membership is already an active member of this project")
	case errors.Is(err, projectmember.ErrProjectMemberNotFound):
		return presenter.RenderError(c, fiber.StatusNotFound, "project_member.not_found", "Project member not found")
	case errors.Is(err, projectmember.ErrProjectNotFound):
		return presenter.RenderError(c, fiber.StatusNotFound, "project.not_found", "Project not found")
	default:
		return presenter.RenderError(c, fiber.StatusInternalServerError, "internal.unexpected", "Internal error")
	}
}
