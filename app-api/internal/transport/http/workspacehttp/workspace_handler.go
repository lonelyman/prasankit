package workspacehttp

import (
	"context"
	"errors"
	"time"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/auth/authsvc"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/internal/modules/workspace/workspaceperm"
	"prasankit-api/internal/modules/workspace/workspacesvc"
	"prasankit-api/internal/transport/http/presenter"

	"github.com/gofiber/fiber/v3"
)

const accountLocalKey = "workspace.account"
const tenantContextLocalKey = "workspace.tenant_context"
const workspaceSlugHeader = "X-Workspace-Slug"

type WorkspaceService interface {
	CheckSlug(ctx context.Context, input workspacesvc.CheckSlugInput) (*workspacesvc.CheckSlugResult, error)
	RegisterWorkspace(ctx context.Context, input workspacesvc.RegisterWorkspaceInput) (*workspacesvc.RegisterWorkspaceResult, error)
	ListMyWorkspaces(ctx context.Context, input workspacesvc.ListMyWorkspacesInput) (*workspacesvc.ListMyWorkspacesResult, error)
	ResolveTenantContext(ctx context.Context, input workspacesvc.ResolveTenantContextInput) (*workspacesvc.ResolveTenantContextResult, error)
}

type SessionService interface {
	CurrentAccount(ctx context.Context, input authsvc.CurrentAccountInput) (*authsvc.CurrentAccountResult, error)
}

type CookieConfig struct {
	Name     string
	TTL      time.Duration
	Secure   bool
	SameSite string
}

type Handler struct {
	workspace WorkspaceService
	session   SessionService
	cookie    CookieConfig
}

type checkSlugResponse struct {
	Slug      string `json:"slug"`
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

type registerWorkspaceRequest struct {
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	ContactEmail string `json:"contact_email"`
}

type registerWorkspaceResponse struct {
	Workspace  workspaceResponse  `json:"workspace"`
	Membership membershipResponse `json:"membership"`
}

type workspaceResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Mode         string `json:"mode"`
	Status       string `json:"status"`
	ContactEmail string `json:"contact_email"`
}

type membershipResponse struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	Role        string `json:"role"`
	Status      string `json:"status"`
}

type myWorkspaceResponse struct {
	Workspace  workspaceResponse  `json:"workspace"`
	Membership membershipResponse `json:"membership"`
}

type currentWorkspaceResponse struct {
	Workspace  workspaceResponse  `json:"workspace"`
	Membership membershipResponse `json:"membership"`
	Context    tenantContextInfo  `json:"context"`
}

type tenantContextInfo struct {
	WorkspaceSlug string `json:"workspace_slug"`
	Role          string `json:"role"`
}

func NewHandler(workspaceService WorkspaceService, sessionService SessionService, cookie CookieConfig) Handler {
	return Handler{
		workspace: workspaceService,
		session:   sessionService,
		cookie:    cookie,
	}
}

func (h Handler) RegisterRoutes(router fiber.Router) {
	workspaces := router.Group("/workspaces")
	workspaces.Get("/check-slug", h.CheckSlug)
	workspaces.Get("/me", h.requireSession, h.MyWorkspaces)
	workspaces.Get("/current", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceView), h.CurrentWorkspace)
	workspaces.Post("/register", h.requireSession, h.RegisterWorkspace)
}

func (h Handler) CheckSlug(c fiber.Ctx) error {
	if h.workspace == nil {
		return h.NotImplemented(c)
	}

	result, err := h.workspace.CheckSlug(c.Context(), workspacesvc.CheckSlugInput{
		Slug: c.Query("slug"),
	})
	if err != nil {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}

	return presenter.RenderItem(c, checkSlugResponse{
		Slug:      result.Slug,
		Available: result.Available,
		Reason:    result.Reason,
	})
}

func (h Handler) RegisterWorkspace(c fiber.Ctx) error {
	if h.workspace == nil {
		return h.NotImplemented(c)
	}

	account, ok := c.Locals(accountLocalKey).(auth.UserAccount)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}

	var req registerWorkspaceRequest
	if err := c.Bind().Body(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Request body is invalid")
	}

	result, err := h.workspace.RegisterWorkspace(c.Context(), workspacesvc.RegisterWorkspaceInput{
		Account:      account,
		Name:         req.Name,
		Slug:         req.Slug,
		ContactEmail: req.ContactEmail,
	})
	if err != nil {
		return renderWorkspaceError(c, err)
	}

	return presenter.RenderItem(c, registerWorkspaceResponse{
		Workspace:  toWorkspaceResponse(result.Workspace),
		Membership: toMembershipResponse(result.Membership),
	}, fiber.StatusCreated)
}

func (h Handler) MyWorkspaces(c fiber.Ctx) error {
	if h.workspace == nil {
		return h.NotImplemented(c)
	}

	account, ok := c.Locals(accountLocalKey).(auth.UserAccount)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}

	query := presenter.ParseOffsetQuery(c)
	result, err := h.workspace.ListMyWorkspaces(c.Context(), workspacesvc.ListMyWorkspacesInput{
		Account: account,
		Limit:   query.Limit,
		Offset:  query.Offset,
	})
	if err != nil {
		return renderWorkspaceError(c, err)
	}

	items := make([]myWorkspaceResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, myWorkspaceResponse{
			Workspace:  toWorkspaceResponse(item.Workspace),
			Membership: toMembershipResponse(item.Membership),
		})
	}

	return presenter.RenderList(c, items, presenter.NewOffsetPagination(result.Total, query.Limit, query.Offset))
}

func (h Handler) CurrentWorkspace(c fiber.Ctx) error {
	result, ok := c.Locals(tenantContextLocalKey).(*workspacesvc.ResolveTenantContextResult)
	if !ok || result == nil {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}

	return presenter.RenderItem(c, currentWorkspaceResponse{
		Workspace:  toWorkspaceResponse(result.Workspace),
		Membership: toMembershipResponse(result.Membership),
		Context: tenantContextInfo{
			WorkspaceSlug: result.Context.WorkspaceSlug,
			Role:          string(result.Context.Role),
		},
	})
}

func (h Handler) requireSession(c fiber.Ctx) error {
	if h.session == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "AUTH_SESSION_REQUIRED", "Authentication session is required")
	}

	result, err := h.session.CurrentAccount(c.Context(), authsvc.CurrentAccountInput{
		SessionToken: c.Cookies(h.cookieName()),
		IPAddress:    c.IP(),
		UserAgent:    c.UserAgent(),
	})
	if err != nil {
		return renderSessionError(c, err)
	}

	c.Locals(accountLocalKey, result.Account)
	return c.Next()
}

func (h Handler) requireTenantContext(c fiber.Ctx) error {
	if h.workspace == nil {
		return h.NotImplemented(c)
	}

	account, ok := c.Locals(accountLocalKey).(auth.UserAccount)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}

	result, err := h.workspace.ResolveTenantContext(c.Context(), workspacesvc.ResolveTenantContextInput{
		Account:       account,
		WorkspaceSlug: c.Get(workspaceSlugHeader),
	})
	if err != nil {
		return renderWorkspaceError(c, err)
	}

	c.Locals(tenantContextLocalKey, result)
	return c.Next()
}

func (h Handler) requireWorkspacePermission(permission workspaceperm.Permission) fiber.Handler {
	return func(c fiber.Ctx) error {
		result, ok := c.Locals(tenantContextLocalKey).(*workspacesvc.ResolveTenantContextResult)
		if !ok || result == nil {
			return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
		}
		if !workspaceperm.Can(result.Context.Role, permission) {
			return presenter.RenderError(c, fiber.StatusForbidden, "PERMISSION_DENIED", "Permission denied")
		}
		return c.Next()
	}
}

func (h Handler) NotImplemented(c fiber.Ctx) error {
	return presenter.RenderError(c, fiber.StatusNotImplemented, "NOT_IMPLEMENTED", "Workspace endpoint is not implemented yet")
}

func (h Handler) cookieName() string {
	if h.cookie.Name == "" {
		return "prasankit_session"
	}
	return h.cookie.Name
}

func renderWorkspaceError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, workspacesvc.ErrWorkspaceNameRequired):
		return presenter.RenderError(c, fiber.StatusBadRequest, "WORKSPACE_NAME_REQUIRED", "Workspace name is required")
	case errors.Is(err, workspacesvc.ErrWorkspaceSlugRequired):
		return presenter.RenderError(c, fiber.StatusBadRequest, "WORKSPACE_SLUG_REQUIRED", "Workspace slug is required")
	case errors.Is(err, workspacesvc.ErrWorkspaceSlugInvalid):
		return presenter.RenderError(c, fiber.StatusBadRequest, "WORKSPACE_SLUG_INVALID", "Workspace slug is invalid")
	case errors.Is(err, workspacesvc.ErrWorkspaceSlugReserved):
		return presenter.RenderError(c, fiber.StatusBadRequest, "WORKSPACE_SLUG_RESERVED", "Workspace slug is reserved")
	case errors.Is(err, workspacesvc.ErrWorkspaceSlugTaken):
		return presenter.RenderError(c, fiber.StatusConflict, "WORKSPACE_SLUG_TAKEN", "Workspace slug is already taken")
	case errors.Is(err, workspacesvc.ErrContactEmailInvalid):
		return presenter.RenderError(c, fiber.StatusBadRequest, "CONTACT_EMAIL_INVALID", "Contact email is invalid")
	case errors.Is(err, workspacesvc.ErrAccountInactive):
		return presenter.RenderError(c, fiber.StatusForbidden, "ACCOUNT_INACTIVE", "Account is inactive")
	case errors.Is(err, workspacesvc.ErrWorkspaceAccessDenied):
		return presenter.RenderError(c, fiber.StatusForbidden, "WORKSPACE_ACCESS_DENIED", "Workspace access denied")
	default:
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
}

func renderSessionError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, authsvc.ErrSessionTokenRequired):
		return presenter.RenderError(c, fiber.StatusUnauthorized, "AUTH_SESSION_REQUIRED", "Authentication session is required")
	case errors.Is(err, authsvc.ErrSessionInvalid):
		return presenter.RenderError(c, fiber.StatusUnauthorized, "AUTH_SESSION_INVALID", "Authentication session is invalid")
	case errors.Is(err, authsvc.ErrSessionExpired):
		return presenter.RenderError(c, fiber.StatusUnauthorized, "AUTH_SESSION_EXPIRED", "Authentication session is expired")
	case errors.Is(err, authsvc.ErrAccountInactive):
		return presenter.RenderError(c, fiber.StatusForbidden, "ACCOUNT_INACTIVE", "Account is inactive")
	default:
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
}

func toWorkspaceResponse(value workspace.Workspace) workspaceResponse {
	return workspaceResponse{
		ID:           value.ID.String(),
		Name:         value.Name,
		Slug:         value.Slug,
		Mode:         string(value.Mode),
		Status:       string(value.Status),
		ContactEmail: value.ContactEmail,
	}
}

func toMembershipResponse(value workspace.Membership) membershipResponse {
	return membershipResponse{
		ID:          value.ID.String(),
		WorkspaceID: value.WorkspaceID.String(),
		Role:        string(value.Role),
		Status:      string(value.Status),
	}
}
