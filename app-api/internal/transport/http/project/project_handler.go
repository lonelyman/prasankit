// Package projecthandler exposes HTTP routes for the project module.
package projecthandler

import (
	"errors"
	"strconv"
	"time"

	"prasankit-api/internal/modules/project"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/internal/transport/http/presenter"
	workspacehandler "prasankit-api/internal/transport/http/workspace"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// Handler handles /workspaces/projects routes.
type Handler struct {
	svc *project.Service
}

// NewHandler constructs a Handler.
func NewHandler(svc *project.Service) *Handler {
	return &Handler{svc: svc}
}

// ── Request types ────────────────────────────────────────────────────────────

type createProjectRequest struct {
	ProjectName       string  `json:"project_name"`
	Slug              string  `json:"slug"`
	ProjectTypeCode   string  `json:"project_type_code"`
	ProjectStatusCode string  `json:"project_status_code"`
	RequestingUnit    *string `json:"requesting_unit"`
	Description       *string `json:"description"`
	StartDate         *string `json:"start_date"` // "YYYY-MM-DD"
	EndDate           *string `json:"end_date"`
}

type updateProjectRequest struct {
	ProjectName     string  `json:"project_name"`
	Slug            string  `json:"slug"`
	ProjectTypeCode string  `json:"project_type_code"`
	RequestingUnit  *string `json:"requesting_unit"`
	Description     *string `json:"description"`
	StartDate       *string `json:"start_date"`
	EndDate         *string `json:"end_date"`
	// NOTE: project_status_code intentionally absent — use POST /status.
}

type changeStatusRequest struct {
	NewStatusCode string `json:"new_status_code"`
}

// ── Response types ────────────────────────────────────────────────────────────

type projectResponse struct {
	ID                   string  `json:"id"`
	WorkspaceID          string  `json:"workspace_id"`
	ProjectName          string  `json:"project_name"`
	Slug                 *string `json:"slug"`
	ProjectTypeCode      string  `json:"project_type_code"`
	ProjectStatusCode    string  `json:"project_status_code"`
	OwnerProjectMemberID *string `json:"owner_project_member_id,omitempty"`
	RequestingUnit       *string `json:"requesting_unit"`
	Description          *string `json:"description"`
	StartDate            *string `json:"start_date"`
	EndDate              *string `json:"end_date"`
	CreatedAt            string  `json:"created_at"`
	UpdatedAt            string  `json:"updated_at"`
}

type paginationResponse struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasMore    bool  `json:"has_more"`
}

type listProjectsResponse struct {
	Items      []projectResponse  `json:"items"`
	Pagination paginationResponse `json:"pagination"`
}

func toProjectResponse(p project.Project) projectResponse {
	resp := projectResponse{
		ID:                p.ID.String(),
		WorkspaceID:       p.WorkspaceID.String(),
		ProjectName:       p.ProjectName,
		Slug:              p.Slug,
		ProjectTypeCode:   p.ProjectTypeCode,
		ProjectStatusCode: p.ProjectStatusCode,
		RequestingUnit:    p.RequestingUnit,
		Description:       p.Description,
		CreatedAt:         p.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:         p.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if p.OwnerProjectMemberID != nil {
		s := p.OwnerProjectMemberID.String()
		resp.OwnerProjectMemberID = &s
	}
	if p.StartDate != nil {
		s := p.StartDate.UTC().Format("2006-01-02")
		resp.StartDate = &s
	}
	if p.EndDate != nil {
		s := p.EndDate.UTC().Format("2006-01-02")
		resp.EndDate = &s
	}
	return resp
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// HandleCreate handles POST /api/v1/workspaces/projects.
func (h *Handler) HandleCreate(c fiber.Ctx) error {
	tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
	if !ok || tc == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}

	var req createProjectRequest
	if err := c.Bind().JSON(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid request body")
	}

	start, fe := parseDate("start_date", req.StartDate)
	if fe != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Validation failed",
			map[string]any{"fields": []project.FieldError{*fe}})
	}
	end, fe := parseDate("end_date", req.EndDate)
	if fe != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Validation failed",
			map[string]any{"fields": []project.FieldError{*fe}})
	}

	p, err := h.svc.CreateProject(c.Context(), project.CreateProjectInput{
		TenantCtx:         *tc,
		ProjectName:       req.ProjectName,
		Slug:              req.Slug,
		ProjectTypeCode:   req.ProjectTypeCode,
		ProjectStatusCode: req.ProjectStatusCode,
		RequestingUnit:    req.RequestingUnit,
		Description:       req.Description,
		StartDate:         start,
		EndDate:           end,
		IP:                c.IP(),
		UserAgent:         string(c.Request().Header.UserAgent()),
		RequestID:         c.Get("X-Request-Id"),
	})
	if err != nil {
		return h.handleServiceError(c, err)
	}
	return presenter.RenderItem(c, toProjectResponse(*p), fiber.StatusCreated)
}

// HandleList handles GET /api/v1/workspaces/projects.
func (h *Handler) HandleList(c fiber.Ctx) error {
	tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
	if !ok || tc == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}

	page, fe := parsePageQuery(c.Query("page"))
	if fe != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Validation failed",
			map[string]any{"fields": []project.FieldError{*fe}})
	}
	limit, fe := parseLimitQuery(c.Query("limit"))
	if fe != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Validation failed",
			map[string]any{"fields": []project.FieldError{*fe}})
	}

	items, total, err := h.svc.ListProjects(c.Context(), project.ListProjectsInput{
		TenantCtx:  *tc,
		Page:       page,
		Limit:      limit,
		StatusCode: c.Query("project_status_code"),
		TypeCode:   c.Query("project_type_code"),
	})
	if err != nil {
		return h.handleServiceError(c, err)
	}

	respItems := make([]projectResponse, 0, len(items))
	for _, p := range items {
		respItems = append(respItems, toProjectResponse(p))
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}
	hasMore := int64(page)*int64(limit) < total

	return presenter.RenderItem(c, listProjectsResponse{
		Items: respItems,
		Pagination: paginationResponse{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
			HasMore:    hasMore,
		},
	})
}

// HandleGet handles GET /api/v1/workspaces/projects/:id.
func (h *Handler) HandleGet(c fiber.Ctx) error {
	tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
	if !ok || tc == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}

	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid project id")
	}

	p, err := h.svc.GetProject(c.Context(), project.GetProjectInput{
		TenantCtx: *tc,
		ProjectID: projectID,
	})
	if err != nil {
		return h.handleServiceError(c, err)
	}
	return presenter.RenderItem(c, toProjectResponse(*p))
}

// HandleUpdate handles PUT /api/v1/workspaces/projects/:id.
func (h *Handler) HandleUpdate(c fiber.Ctx) error {
	tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
	if !ok || tc == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}

	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid project id")
	}

	var req updateProjectRequest
	if err := c.Bind().JSON(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid request body")
	}

	start, fe := parseDate("start_date", req.StartDate)
	if fe != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Validation failed",
			map[string]any{"fields": []project.FieldError{*fe}})
	}
	end, fe := parseDate("end_date", req.EndDate)
	if fe != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Validation failed",
			map[string]any{"fields": []project.FieldError{*fe}})
	}

	p, err := h.svc.UpdateProject(c.Context(), project.UpdateProjectInput{
		TenantCtx:       *tc,
		ProjectID:       projectID,
		ProjectName:     req.ProjectName,
		Slug:            req.Slug,
		ProjectTypeCode: req.ProjectTypeCode,
		RequestingUnit:  req.RequestingUnit,
		Description:     req.Description,
		StartDate:       start,
		EndDate:         end,
		IP:              c.IP(),
		UserAgent:       string(c.Request().Header.UserAgent()),
		RequestID:       c.Get("X-Request-Id"),
	})
	if err != nil {
		return h.handleServiceError(c, err)
	}
	return presenter.RenderItem(c, toProjectResponse(*p))
}

// HandleDelete handles DELETE /api/v1/workspaces/projects/:id.
// Returns true 204 (empty body) on success.
func (h *Handler) HandleDelete(c fiber.Ctx) error {
	tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
	if !ok || tc == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}

	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid project id")
	}

	err = h.svc.DeleteProject(c.Context(), project.DeleteProjectInput{
		TenantCtx: *tc,
		ProjectID: projectID,
		IP:        c.IP(),
		UserAgent: string(c.Request().Header.UserAgent()),
		RequestID: c.Get("X-Request-Id"),
	})
	if err != nil {
		return h.handleServiceError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// HandleChangeStatus handles POST /api/v1/workspaces/projects/:id/status.
func (h *Handler) HandleChangeStatus(c fiber.Ctx) error {
	tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
	if !ok || tc == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}

	projectID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid project id")
	}

	var req changeStatusRequest
	if err := c.Bind().JSON(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid request body")
	}

	p, err := h.svc.ChangeStatus(c.Context(), project.ChangeStatusInput{
		TenantCtx:     *tc,
		ProjectID:     projectID,
		NewStatusCode: req.NewStatusCode,
		IP:            c.IP(),
		UserAgent:     string(c.Request().Header.UserAgent()),
		RequestID:     c.Get("X-Request-Id"),
	})
	if err != nil {
		return h.handleServiceError(c, err)
	}
	return presenter.RenderItem(c, toProjectResponse(*p))
}

// ── helpers ────────────────────────────────────────────────────────────────────

// parseDate parses a YYYY-MM-DD string into UTC midnight time.Time.
//
// INVARIANT (load-bearing): the returned *time.Time is always at UTC midnight
// (time.Parse("2006-01-02", s) guarantees this per Go stdlib). Service-layer
// start_date <= end_date comparison and DB ck_projects_date_range agree only
// because both sides are UTC midnight. NEVER use time.Now() / time.Local /
// arbitrary time.Time for date logic in this module — that would introduce
// timezone drift between Go comparison and Postgres DATE storage.
func parseDate(field string, s *string) (*time.Time, *project.FieldError) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil, &project.FieldError{Field: field, Message: "must be a YYYY-MM-DD date"}
	}
	return &t, nil
}

// parsePageQuery validates the `page` query parameter.
// empty → 1; non-numeric → field error; < 1 → field error.
func parsePageQuery(q string) (int, *project.FieldError) {
	if q == "" {
		return 1, nil
	}
	n, err := strconv.Atoi(q)
	if err != nil {
		return 0, &project.FieldError{Field: "page", Message: "must be a positive integer"}
	}
	if n < 1 {
		return 0, &project.FieldError{Field: "page", Message: "must be >= 1"}
	}
	return n, nil
}

// parseLimitQuery validates the `limit` query parameter.
// empty → 10; non-numeric → field error; < 1 or > 100 → field error.
func parseLimitQuery(q string) (int, *project.FieldError) {
	if q == "" {
		return 10, nil
	}
	n, err := strconv.Atoi(q)
	if err != nil {
		return 0, &project.FieldError{Field: "limit", Message: "must be an integer in [1,100]"}
	}
	if n < 1 || n > 100 {
		return 0, &project.FieldError{Field: "limit", Message: "must be between 1 and 100"}
	}
	return n, nil
}

// handleServiceError maps project service errors to HTTP responses.
func (h *Handler) handleServiceError(c fiber.Ctx, err error) error {
	var valErr *project.ValidationError
	switch {
	case errors.As(err, &valErr):
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Validation failed",
			map[string]any{"fields": valErr.Fields})
	case errors.Is(err, project.ErrSlugTaken):
		return presenter.RenderError(c, fiber.StatusConflict, "project.slug_taken", "Slug is already taken")
	case errors.Is(err, project.ErrInvalidStatusCode):
		return presenter.RenderError(c, fiber.StatusUnprocessableEntity, "project.invalid_master_code", "Invalid project_status_code",
			map[string]any{"field": "project_status_code"})
	case errors.Is(err, project.ErrInvalidTypeCode):
		return presenter.RenderError(c, fiber.StatusUnprocessableEntity, "project.invalid_master_code", "Invalid project_type_code",
			map[string]any{"field": "project_type_code"})
	case errors.Is(err, project.ErrProjectNotFound):
		return presenter.RenderError(c, fiber.StatusNotFound, "project.not_found", "Project not found")
	default:
		return presenter.RenderError(c, fiber.StatusInternalServerError, "internal.unexpected", "Internal error")
	}
}
