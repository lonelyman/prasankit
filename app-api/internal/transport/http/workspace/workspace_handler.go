package workspacehandler

import (
	"errors"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/workspace"
	authhandler "prasankit-api/internal/transport/http/auth"
	"prasankit-api/internal/transport/http/presenter"

	"github.com/gofiber/fiber/v3"
)

// LocalsKeyTenant is the c.Locals key set by requireTenantContext middleware.
const LocalsKeyTenant = "tenant_context"

// Handler handles /workspaces routes.
type Handler struct {
	svc *workspace.Service
}

// NewHandler constructs a Handler.
func NewHandler(svc *workspace.Service) *Handler {
	return &Handler{svc: svc}
}

// ── Response types ────────────────────────────────────────────────────────────

type workspaceResponse struct {
	ID                  string `json:"id"`
	WorkspaceName       string `json:"workspace_name"`
	Slug                string `json:"slug"`
	WorkspaceStatusCode string `json:"workspace_status_code"`
	ContactEmail        string `json:"contact_email"`
}

type workspaceWithRoleResponse struct {
	ID                  string `json:"id"`
	WorkspaceName       string `json:"workspace_name"`
	Slug                string `json:"slug"`
	WorkspaceStatusCode string `json:"workspace_status_code"`
	ContactEmail        string `json:"contact_email"`
	OrgRoleCode         string `json:"org_role_code"`
}

type currentWorkspaceResponse struct {
	ID                  string `json:"id"`
	WorkspaceName       string `json:"workspace_name"`
	Slug                string `json:"slug"`
	WorkspaceStatusCode string `json:"workspace_status_code"`
	OrgRoleCode         string `json:"org_role_code"`
}

type memberPickerResponse struct {
	WorkspaceMembershipID string `json:"workspace_membership_id"`
	DisplayName           string `json:"display_name"`
	OrgRoleCode           string `json:"org_role_code"`
}

type listMembersResponse struct {
	Items []memberPickerResponse `json:"items"`
	Count int                    `json:"count"`
}

func toWorkspaceResponse(ws workspace.Workspace) workspaceResponse {
	return workspaceResponse{
		ID:                  ws.ID.String(),
		WorkspaceName:       ws.WorkspaceName,
		Slug:                ws.Slug,
		WorkspaceStatusCode: ws.WorkspaceStatusCode,
		ContactEmail:        ws.ContactEmail,
	}
}

func toWorkspaceWithRoleResponse(wr workspace.WorkspaceWithRole) workspaceWithRoleResponse {
	return workspaceWithRoleResponse{
		ID:                  wr.Workspace.ID.String(),
		WorkspaceName:       wr.Workspace.WorkspaceName,
		Slug:                wr.Workspace.Slug,
		WorkspaceStatusCode: wr.Workspace.WorkspaceStatusCode,
		ContactEmail:        wr.Workspace.ContactEmail,
		OrgRoleCode:         wr.OrgRoleCode,
	}
}

// ── Request types ─────────────────────────────────────────────────────────────

type createWorkspaceRequest struct {
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	ContactEmail string `json:"contact_email"`
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// HandleCreate handles POST /workspaces (requireSession).
func (h *Handler) HandleCreate(c fiber.Ctx) error {
	account, ok := c.Locals(authhandler.LocalsKeyAccount).(*auth.Account)
	if !ok || account == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}

	var req createWorkspaceRequest
	if err := c.Bind().JSON(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid request body")
	}

	ws, err := h.svc.CreateWorkspace(c.Context(), workspace.CreateWorkspaceInput{
		AccountID:    account.ID,
		Name:         req.Name,
		Slug:         req.Slug,
		ContactEmail: req.ContactEmail,
		IP:           c.IP(),
		UserAgent:    string(c.Request().Header.UserAgent()),
		RequestID:    c.Get("X-Request-Id"),
	})
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return presenter.RenderItem(c, toWorkspaceResponse(*ws), fiber.StatusCreated)
}

// HandleList handles GET /workspaces (requireSession).
func (h *Handler) HandleList(c fiber.Ctx) error {
	account, ok := c.Locals(authhandler.LocalsKeyAccount).(*auth.Account)
	if !ok || account == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}

	items, err := h.svc.ListMyWorkspaces(c.Context(), account.ID)
	if err != nil {
		return err
	}

	resp := make([]workspaceWithRoleResponse, len(items))
	for i, item := range items {
		resp[i] = toWorkspaceWithRoleResponse(item)
	}
	return presenter.RenderItem(c, map[string]any{"items": resp})
}

// HandleGetCurrent handles GET /workspaces/current (requireSession + requireTenantContext).
func (h *Handler) HandleGetCurrent(c fiber.Ctx) error {
	tc, ok := c.Locals(LocalsKeyTenant).(*workspace.TenantContext)
	if !ok || tc == nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "tenant.workspace_required", "Workspace context not resolved")
	}

	item, err := h.svc.GetCurrentWorkspace(c.Context(), *tc)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return presenter.RenderItem(c, currentWorkspaceResponse{
		ID:                  item.Workspace.ID.String(),
		WorkspaceName:       item.Workspace.WorkspaceName,
		Slug:                item.Workspace.Slug,
		WorkspaceStatusCode: item.Workspace.WorkspaceStatusCode,
		OrgRoleCode:         tc.OrgRoleCode,
	})
}

// HandleListMembers handles GET /workspaces/members
// (requireSession + requireTenantContext + requireWorkspacePermission(invite)).
// Returns the active workspace memberships with display_name for the add-member picker.
func (h *Handler) HandleListMembers(c fiber.Ctx) error {
	tc, ok := c.Locals(LocalsKeyTenant).(*workspace.TenantContext)
	if !ok || tc == nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "tenant.workspace_required", "Workspace context not resolved")
	}

	members, err := h.svc.ListWorkspaceMembers(c.Context(), *tc)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	items := make([]memberPickerResponse, 0, len(members))
	for _, m := range members {
		items = append(items, memberPickerResponse{
			WorkspaceMembershipID: m.ID.String(),
			DisplayName:           m.DisplayName,
			OrgRoleCode:           m.OrgRoleCode,
		})
	}
	return presenter.RenderItem(c, listMembersResponse{Items: items, Count: len(items)})
}

// handleServiceError maps service errors to HTTP responses.
func (h *Handler) handleServiceError(c fiber.Ctx, err error) error {
	var valErr *workspace.ValidationError
	if errors.As(err, &valErr) {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Validation failed", valErr.Fields)
	}
	if errors.Is(err, workspace.ErrSlugReserved) {
		return presenter.RenderError(c, fiber.StatusBadRequest, "workspace.slug_reserved", "Slug is reserved and cannot be used")
	}
	if errors.Is(err, workspace.ErrSlugTaken) {
		return presenter.RenderError(c, fiber.StatusConflict, "workspace.slug_taken", "Slug is already taken")
	}
	if errors.Is(err, workspace.ErrWorkspaceNotFound) {
		return presenter.RenderError(c, fiber.StatusNotFound, "tenant.workspace_not_found", "Workspace not found")
	}
	if errors.Is(err, workspace.ErrForbidden) {
		return presenter.RenderError(c, fiber.StatusForbidden, "tenant.forbidden", "Access denied to this workspace")
	}
	return err
}
