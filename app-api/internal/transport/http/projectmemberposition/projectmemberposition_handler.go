// Package projectmemberpositionhandler exposes HTTP routes for the project_member_position
// (junction) module.
package projectmemberpositionhandler

import (
	"errors"
	"time"

	"prasankit-api/internal/modules/projectmemberposition"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/internal/transport/http/presenter"
	workspacehandler "prasankit-api/internal/transport/http/workspace"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// Handler handles /workspaces/projects/:id/members/:memberId/positions routes.
type Handler struct {
	svc *projectmemberposition.Service
}

// NewHandler constructs a Handler.
func NewHandler(svc *projectmemberposition.Service) *Handler {
	return &Handler{svc: svc}
}

// ── Request types ────────────────────────────────────────────────────────────

type assignRequest struct {
	ProjectPositionCode string `json:"project_position_code"`
}

// ── Response types ────────────────────────────────────────────────────────────

type assignmentResponse struct {
	ID                  string `json:"id"`
	WorkspaceID         string `json:"workspace_id"`
	ProjectMemberID     string `json:"project_member_id"`
	ProjectPositionCode string `json:"project_position_code"`
	CreatedAt           string `json:"created_at"`
}

type positionWithLabelResponse struct {
	ID                  string `json:"id"`
	WorkspaceID         string `json:"workspace_id"`
	ProjectMemberID     string `json:"project_member_id"`
	ProjectPositionCode string `json:"project_position_code"`
	LabelTH             string `json:"label_th"`
	LabelEN             string `json:"label_en"`
	Status              string `json:"status"`
	CreatedAt           string `json:"created_at"`
}

type listResponse struct {
	Items []positionWithLabelResponse `json:"items"`
	Count int                         `json:"count"`
}

func toAssignmentResponse(mp projectmemberposition.ProjectMemberPosition) assignmentResponse {
	return assignmentResponse{
		ID:                  mp.ID.String(),
		WorkspaceID:         mp.WorkspaceID.String(),
		ProjectMemberID:     mp.ProjectMemberID.String(),
		ProjectPositionCode: mp.ProjectPositionCode,
		CreatedAt:           mp.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func toPositionWithLabelResponse(p projectmemberposition.PositionWithLabel) positionWithLabelResponse {
	return positionWithLabelResponse{
		ID:                  p.ID.String(),
		WorkspaceID:         p.WorkspaceID.String(),
		ProjectMemberID:     p.ProjectMemberID.String(),
		ProjectPositionCode: p.ProjectPositionCode,
		LabelTH:             p.LabelTH,
		LabelEN:             p.LabelEN,
		Status:              p.Status,
		CreatedAt:           p.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// HandleAssign handles POST /api/v1/workspaces/projects/:id/members/:memberId/positions.
func (h *Handler) HandleAssign(c fiber.Ctx) error {
	tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
	if !ok || tc == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}

	projectID, memberID, err := parseIDs(c)
	if err != nil {
		return err
	}

	var req assignRequest
	if err := c.Bind().JSON(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid request body")
	}

	mp, svcErr := h.svc.Assign(c.Context(), projectmemberposition.AssignInput{
		TenantCtx:           *tc,
		ProjectID:           projectID,
		ProjectMemberID:     memberID,
		ProjectPositionCode: req.ProjectPositionCode,
		IP:                  c.IP(),
		UserAgent:           string(c.Request().Header.UserAgent()),
		RequestID:           c.Get("X-Request-Id"),
	})
	if svcErr != nil {
		return h.handleServiceError(c, svcErr)
	}
	return presenter.RenderItem(c, toAssignmentResponse(*mp), fiber.StatusCreated)
}

// HandleList handles GET /api/v1/workspaces/projects/:id/members/:memberId/positions.
func (h *Handler) HandleList(c fiber.Ctx) error {
	tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
	if !ok || tc == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}

	projectID, memberID, err := parseIDs(c)
	if err != nil {
		return err
	}

	items, svcErr := h.svc.List(c.Context(), projectmemberposition.ListInput{
		TenantCtx:       *tc,
		ProjectID:       projectID,
		ProjectMemberID: memberID,
	})
	if svcErr != nil {
		return h.handleServiceError(c, svcErr)
	}

	respItems := make([]positionWithLabelResponse, 0, len(items))
	for _, p := range items {
		respItems = append(respItems, toPositionWithLabelResponse(p))
	}
	return presenter.RenderItem(c, listResponse{Items: respItems, Count: len(respItems)})
}

// HandleUnassign handles DELETE /api/v1/workspaces/projects/:id/members/:memberId/positions/:code.
// Returns 204 (empty body) on success.
func (h *Handler) HandleUnassign(c fiber.Ctx) error {
	tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
	if !ok || tc == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}

	projectID, memberID, err := parseIDs(c)
	if err != nil {
		return err
	}
	code := c.Params("code")
	if code == "" {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid code")
	}

	svcErr := h.svc.Unassign(c.Context(), projectmemberposition.UnassignInput{
		TenantCtx:           *tc,
		ProjectID:           projectID,
		ProjectMemberID:     memberID,
		ProjectPositionCode: code,
		IP:                  c.IP(),
		UserAgent:           string(c.Request().Header.UserAgent()),
		RequestID:           c.Get("X-Request-Id"),
	})
	if svcErr != nil {
		return h.handleServiceError(c, svcErr)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ── helpers ────────────────────────────────────────────────────────────────────

// parseIDs parses :id (projectID) and :memberId (memberID) path params, returning a rendered
// 400 error when either is malformed.
func parseIDs(c fiber.Ctx) (uuid.UUID, uuid.UUID, error) {
	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid project id")
	}
	memberID, err := uuid.Parse(c.Params("memberId"))
	if err != nil {
		return uuid.Nil, uuid.Nil, presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid member id")
	}
	return projectID, memberID, nil
}

// handleServiceError maps projectmemberposition service errors to HTTP responses.
func (h *Handler) handleServiceError(c fiber.Ctx, err error) error {
	var valErr *projectmemberposition.ValidationError
	switch {
	case errors.As(err, &valErr):
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Validation failed",
			map[string]any{"fields": valErr.Fields})
	case errors.Is(err, projectmemberposition.ErrAlreadyAssigned):
		return presenter.RenderError(c, fiber.StatusConflict, "project_member_position.already_assigned", "Position already assigned")
	case errors.Is(err, projectmemberposition.ErrInvalidPositionCode):
		return presenter.RenderError(c, fiber.StatusUnprocessableEntity, "project_member_position.invalid_position_code", "Invalid project_position_code",
			map[string]any{"field": "project_position_code"})
	case errors.Is(err, projectmemberposition.ErrMemberRemoved):
		return presenter.RenderError(c, fiber.StatusUnprocessableEntity, "project_member_position.member_removed", "Project member is removed")
	case errors.Is(err, projectmemberposition.ErrProjectMemberNotFound):
		return presenter.RenderError(c, fiber.StatusNotFound, "project_member_position.member_not_found", "Project member not found")
	case errors.Is(err, projectmemberposition.ErrAssignmentNotFound):
		return presenter.RenderError(c, fiber.StatusNotFound, "project_member_position.not_found", "Assignment not found")
	case errors.Is(err, projectmemberposition.ErrProjectNotFound):
		return presenter.RenderError(c, fiber.StatusNotFound, "project.not_found", "Project not found")
	default:
		return presenter.RenderError(c, fiber.StatusInternalServerError, "internal.unexpected", "Internal error")
	}
}
