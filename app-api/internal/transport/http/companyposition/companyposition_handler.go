// Package companypositionhandler exposes HTTP routes for the company_position module
// (master CRUD + the ws company_position attach, D41).
package companypositionhandler

import (
	"errors"
	"strconv"
	"time"

	"prasankit-api/internal/modules/companyposition"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/internal/transport/http/presenter"
	workspacehandler "prasankit-api/internal/transport/http/workspace"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// Handler handles /workspaces/company-positions + /workspaces/memberships/:id/company-position routes.
type Handler struct {
	svc *companyposition.Service
}

// NewHandler constructs a Handler.
func NewHandler(svc *companyposition.Service) *Handler {
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
	Code        string  `json:"code"`
	LabelTH     string  `json:"label_th"`
	LabelEN     string  `json:"label_en"`
	Description *string `json:"description"`
	SortOrder   int     `json:"sort_order"`
	Status      string  `json:"status"`
}

// setCompanyPositionRequest carries the nullable code for the ws-attach (set/clear share this).
// company_position_code = null (or absent) -> clear.
type setCompanyPositionRequest struct {
	CompanyPositionCode *string `json:"company_position_code"`
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

func toResp(p companyposition.CompanyPosition) positionResponse {
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

// ── Master Handlers ─────────────────────────────────────────────────────────────

// HandleCreate handles POST /api/v1/workspaces/company-positions.
func (h *Handler) HandleCreate(c fiber.Ctx) error {
	tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
	if !ok || tc == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}

	var req createRequest
	if err := c.Bind().JSON(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid request body")
	}

	p, err := h.svc.Create(c.Context(), companyposition.CreateInput{
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

// HandleList handles GET /api/v1/workspaces/company-positions.
func (h *Handler) HandleList(c fiber.Ctx) error {
	tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
	if !ok || tc == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}

	page, fe := parsePageQuery(c.Query("page"))
	if fe != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Validation failed",
			map[string]any{"fields": []companyposition.FieldError{*fe}})
	}
	limit, fe := parseLimitQuery(c.Query("limit"))
	if fe != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Validation failed",
			map[string]any{"fields": []companyposition.FieldError{*fe}})
	}

	items, total, err := h.svc.List(c.Context(), companyposition.ListInput{
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

// HandleUpdate handles PUT /api/v1/workspaces/company-positions/:code.
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

	p, err := h.svc.Update(c.Context(), companyposition.UpdateInput{
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

// HandleDeprecate handles POST /api/v1/workspaces/company-positions/:code/deprecate.
func (h *Handler) HandleDeprecate(c fiber.Ctx) error {
	tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
	if !ok || tc == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}

	code := c.Params("code")
	if code == "" {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid code")
	}

	p, err := h.svc.Deprecate(c.Context(), companyposition.DeprecateInput{
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

// ── ws company_position attach (D41) ──────────────────────────────────────────

// HandleSetCompanyPosition handles PUT /api/v1/workspaces/memberships/:membershipId/company-position.
// company_position_code = null (or absent) -> clear. Returns 200 with no body item.
func (h *Handler) HandleSetCompanyPosition(c fiber.Ctx) error {
	tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
	if !ok || tc == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}

	membershipID, err := uuid.Parse(c.Params("membershipId"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid membership id")
	}

	var req setCompanyPositionRequest
	if err := c.Bind().JSON(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid request body")
	}

	if err := h.svc.SetCompanyPosition(c.Context(), companyposition.SetCompanyPositionInput{
		TenantCtx:           *tc,
		MembershipID:        membershipID,
		CompanyPositionCode: req.CompanyPositionCode,
		IP:                  c.IP(),
		UserAgent:           string(c.Request().Header.UserAgent()),
		RequestID:           c.Get("X-Request-Id"),
	}); err != nil {
		return h.handleServiceError(c, err)
	}

	var codeResp *string
	if req.CompanyPositionCode != nil && *req.CompanyPositionCode != "" {
		codeResp = req.CompanyPositionCode
	}
	return presenter.RenderItem(c, map[string]any{
		"membership_id":         membershipID.String(),
		"company_position_code": codeResp,
	})
}

// ── helpers ────────────────────────────────────────────────────────────────────

func parsePageQuery(q string) (int, *companyposition.FieldError) {
	if q == "" {
		return 1, nil
	}
	n, err := strconv.Atoi(q)
	if err != nil {
		return 0, &companyposition.FieldError{Field: "page", Message: "must be a positive integer"}
	}
	if n < 1 {
		return 0, &companyposition.FieldError{Field: "page", Message: "must be >= 1"}
	}
	return n, nil
}

func parseLimitQuery(q string) (int, *companyposition.FieldError) {
	if q == "" {
		return 10, nil
	}
	n, err := strconv.Atoi(q)
	if err != nil {
		return 0, &companyposition.FieldError{Field: "limit", Message: "must be an integer in [1,100]"}
	}
	if n < 1 || n > 100 {
		return 0, &companyposition.FieldError{Field: "limit", Message: "must be between 1 and 100"}
	}
	return n, nil
}

// handleServiceError maps companyposition service errors to HTTP responses.
func (h *Handler) handleServiceError(c fiber.Ctx, err error) error {
	var valErr *companyposition.ValidationError
	switch {
	case errors.As(err, &valErr):
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Validation failed",
			map[string]any{"fields": valErr.Fields})
	case errors.Is(err, companyposition.ErrCodeTaken):
		return presenter.RenderError(c, fiber.StatusConflict, "company_position.code_taken", "Code already exists in this workspace",
			map[string]any{"field": "code"})
	case errors.Is(err, companyposition.ErrCodeImmutable):
		return presenter.RenderError(c, fiber.StatusUnprocessableEntity, "company_position.code_immutable", "Code is immutable",
			map[string]any{"field": "code"})
	case errors.Is(err, companyposition.ErrSystemImmutable):
		return presenter.RenderError(c, fiber.StatusUnprocessableEntity, "company_position.system_immutable", "System position cannot be modified")
	case errors.Is(err, companyposition.ErrInvalidStatus):
		return presenter.RenderError(c, fiber.StatusUnprocessableEntity, "company_position.invalid_status", "Invalid status",
			map[string]any{"field": "status"})
	case errors.Is(err, companyposition.ErrInvalidPositionCode):
		return presenter.RenderError(c, fiber.StatusUnprocessableEntity, "company_position.invalid_position_code", "Invalid company_position_code",
			map[string]any{"field": "company_position_code"})
	case errors.Is(err, companyposition.ErrMembershipNotFound):
		return presenter.RenderError(c, fiber.StatusNotFound, "company_position.membership_not_found", "Membership not found")
	case errors.Is(err, companyposition.ErrPositionNotFound):
		return presenter.RenderError(c, fiber.StatusNotFound, "company_position.not_found", "Company position not found")
	default:
		return presenter.RenderError(c, fiber.StatusInternalServerError, "internal.unexpected", "Internal error")
	}
}
