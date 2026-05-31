// Package projectpositionhandler exposes HTTP routes for the project_position module.
package projectpositionhandler

import (
	"errors"
	"strconv"
	"time"

	"prasankit-api/internal/modules/projectposition"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/internal/transport/http/presenter"
	workspacehandler "prasankit-api/internal/transport/http/workspace"

	"github.com/gofiber/fiber/v3"
)

// Handler handles /workspaces/project-positions routes.
type Handler struct {
	svc *projectposition.Service
}

// NewHandler constructs a Handler.
func NewHandler(svc *projectposition.Service) *Handler {
	return &Handler{svc: svc}
}

// ── Request types ────────────────────────────────────────────────────────────

type createRequest struct {
	Code        string  `json:"code"`
	LabelTH     string  `json:"label_th"`
	LabelEN     string  `json:"label_en"`
	Description *string `json:"description"`
	SortOrder   int     `json:"sort_order"`
}

type updateRequest struct {
	Code        string  `json:"code"` // optional; if present must equal path code (else 422 immutable)
	LabelTH     string  `json:"label_th"`
	LabelEN     string  `json:"label_en"`
	Description *string `json:"description"`
	SortOrder   int     `json:"sort_order"`
	Status      string  `json:"status"`
}

// ── Response types ────────────────────────────────────────────────────────────

type positionResponse struct {
	ID          string  `json:"id"`
	WorkspaceID string  `json:"workspace_id"`
	Code        string  `json:"code"`
	LabelTH     string  `json:"label_th"`
	LabelEN     string  `json:"label_en"`
	Description *string `json:"description"`
	SortOrder   int     `json:"sort_order"`
	IsSystem    bool    `json:"is_system"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type paginationResponse struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasMore    bool  `json:"has_more"`
}

type listResponse struct {
	Items      []positionResponse `json:"items"`
	Pagination paginationResponse `json:"pagination"`
}

func toResp(p projectposition.ProjectPosition) positionResponse {
	return positionResponse{
		ID:          p.ID.String(),
		WorkspaceID: p.WorkspaceID.String(),
		Code:        p.Code,
		LabelTH:     p.LabelTH,
		LabelEN:     p.LabelEN,
		Description: p.Description,
		SortOrder:   p.SortOrder,
		IsSystem:    p.IsSystem,
		Status:      p.Status,
		CreatedAt:   p.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   p.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// HandleCreate handles POST /api/v1/workspaces/project-positions.
func (h *Handler) HandleCreate(c fiber.Ctx) error {
	tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
	if !ok || tc == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}

	var req createRequest
	if err := c.Bind().JSON(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid request body")
	}

	p, err := h.svc.Create(c.Context(), projectposition.CreateInput{
		TenantCtx:   *tc,
		Code:        req.Code,
		LabelTH:     req.LabelTH,
		LabelEN:     req.LabelEN,
		Description: req.Description,
		SortOrder:   req.SortOrder,
		IP:          c.IP(),
		UserAgent:   string(c.Request().Header.UserAgent()),
		RequestID:   c.Get("X-Request-Id"),
	})
	if err != nil {
		return h.handleServiceError(c, err)
	}
	return presenter.RenderItem(c, toResp(*p), fiber.StatusCreated)
}

// HandleList handles GET /api/v1/workspaces/project-positions.
func (h *Handler) HandleList(c fiber.Ctx) error {
	tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
	if !ok || tc == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}

	page, fe := parsePageQuery(c.Query("page"))
	if fe != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Validation failed",
			map[string]any{"fields": []projectposition.FieldError{*fe}})
	}
	limit, fe := parseLimitQuery(c.Query("limit"))
	if fe != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Validation failed",
			map[string]any{"fields": []projectposition.FieldError{*fe}})
	}

	items, total, err := h.svc.List(c.Context(), projectposition.ListInput{
		TenantCtx: *tc,
		Page:      page,
		Limit:     limit,
		Status:    c.Query("status"),
	})
	if err != nil {
		return h.handleServiceError(c, err)
	}

	respItems := make([]positionResponse, 0, len(items))
	for _, p := range items {
		respItems = append(respItems, toResp(p))
	}
	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}
	hasMore := int64(page)*int64(limit) < total

	return presenter.RenderItem(c, listResponse{
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

// HandleUpdate handles PUT /api/v1/workspaces/project-positions/:code.
func (h *Handler) HandleUpdate(c fiber.Ctx) error {
	tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
	if !ok || tc == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}

	code := c.Params("code")
	if code == "" {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid code")
	}

	var req updateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid request body")
	}

	p, err := h.svc.Update(c.Context(), projectposition.UpdateInput{
		TenantCtx:   *tc,
		Code:        code,
		BodyCode:    req.Code,
		LabelTH:     req.LabelTH,
		LabelEN:     req.LabelEN,
		Description: req.Description,
		SortOrder:   req.SortOrder,
		Status:      req.Status,
		IP:          c.IP(),
		UserAgent:   string(c.Request().Header.UserAgent()),
		RequestID:   c.Get("X-Request-Id"),
	})
	if err != nil {
		return h.handleServiceError(c, err)
	}
	return presenter.RenderItem(c, toResp(*p))
}

// HandleDeprecate handles POST /api/v1/workspaces/project-positions/:code/deprecate.
func (h *Handler) HandleDeprecate(c fiber.Ctx) error {
	tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
	if !ok || tc == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}

	code := c.Params("code")
	if code == "" {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid code")
	}

	p, err := h.svc.Deprecate(c.Context(), projectposition.DeprecateInput{
		TenantCtx: *tc,
		Code:      code,
		IP:        c.IP(),
		UserAgent: string(c.Request().Header.UserAgent()),
		RequestID: c.Get("X-Request-Id"),
	})
	if err != nil {
		return h.handleServiceError(c, err)
	}
	return presenter.RenderItem(c, toResp(*p))
}

// ── helpers ────────────────────────────────────────────────────────────────────

func parsePageQuery(q string) (int, *projectposition.FieldError) {
	if q == "" {
		return 1, nil
	}
	n, err := strconv.Atoi(q)
	if err != nil {
		return 0, &projectposition.FieldError{Field: "page", Message: "must be a positive integer"}
	}
	if n < 1 {
		return 0, &projectposition.FieldError{Field: "page", Message: "must be >= 1"}
	}
	return n, nil
}

func parseLimitQuery(q string) (int, *projectposition.FieldError) {
	if q == "" {
		return 10, nil
	}
	n, err := strconv.Atoi(q)
	if err != nil {
		return 0, &projectposition.FieldError{Field: "limit", Message: "must be an integer in [1,100]"}
	}
	if n < 1 || n > 100 {
		return 0, &projectposition.FieldError{Field: "limit", Message: "must be between 1 and 100"}
	}
	return n, nil
}

// handleServiceError maps projectposition service errors to HTTP responses.
func (h *Handler) handleServiceError(c fiber.Ctx, err error) error {
	var valErr *projectposition.ValidationError
	switch {
	case errors.As(err, &valErr):
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Validation failed",
			map[string]any{"fields": valErr.Fields})
	case errors.Is(err, projectposition.ErrCodeTaken):
		return presenter.RenderError(c, fiber.StatusConflict, "project_position.code_taken", "Code already exists in this workspace",
			map[string]any{"field": "code"})
	case errors.Is(err, projectposition.ErrCodeImmutable):
		return presenter.RenderError(c, fiber.StatusUnprocessableEntity, "project_position.code_immutable", "Code is immutable",
			map[string]any{"field": "code"})
	case errors.Is(err, projectposition.ErrSystemImmutable):
		return presenter.RenderError(c, fiber.StatusUnprocessableEntity, "project_position.system_immutable", "System position cannot be modified")
	case errors.Is(err, projectposition.ErrInvalidStatus):
		return presenter.RenderError(c, fiber.StatusUnprocessableEntity, "project_position.invalid_status", "Invalid status",
			map[string]any{"field": "status"})
	case errors.Is(err, projectposition.ErrPositionNotFound):
		return presenter.RenderError(c, fiber.StatusNotFound, "project_position.not_found", "Project position not found")
	default:
		return presenter.RenderError(c, fiber.StatusInternalServerError, "internal.unexpected", "Internal error")
	}
}
