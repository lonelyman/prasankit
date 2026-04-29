package projecthttp

import (
	"context"
	"errors"
	"time"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/auth/authsvc"
	"prasankit-api/internal/modules/project"
	"prasankit-api/internal/modules/project/projectsvc"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/internal/modules/workspace/workspaceperm"
	"prasankit-api/internal/modules/workspace/workspacesvc"
	"prasankit-api/internal/transport/http/presenter"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const accountLocalKey = "project.account"
const tenantContextLocalKey = "project.tenant_context"
const workspaceSlugHeader = "X-Workspace-Slug"

type ProjectService interface {
	CreateProject(ctx context.Context, input projectsvc.CreateProjectInput) (*projectsvc.CreateProjectResult, error)
	ListProjects(ctx context.Context, input projectsvc.ListProjectsInput) (*projectsvc.ListProjectsResult, error)
	GetProject(ctx context.Context, input projectsvc.GetProjectInput) (*projectsvc.GetProjectResult, error)
	UpdateProject(ctx context.Context, input projectsvc.UpdateProjectInput) (*projectsvc.UpdateProjectResult, error)
}

type SessionService interface {
	CurrentAccount(ctx context.Context, input authsvc.CurrentAccountInput) (*authsvc.CurrentAccountResult, error)
}

type TenantResolver interface {
	ResolveTenantContext(ctx context.Context, input workspacesvc.ResolveTenantContextInput) (*workspacesvc.ResolveTenantContextResult, error)
}

type CookieConfig struct {
	Name     string
	TTL      time.Duration
	Secure   bool
	SameSite string
}

type Handler struct {
	projects ProjectService
	session  SessionService
	tenant   TenantResolver
	cookie   CookieConfig
}

type createProjectRequest struct {
	Name                   string `json:"name"`
	Type                   string `json:"type"`
	Description            string `json:"description"`
	ClientOrRequestingUnit string `json:"client_or_requesting_unit"`
	ScopeOrObjective       string `json:"scope_or_objective"`
}

type updateProjectRequest struct {
	Name                   *string `json:"name"`
	Type                   *string `json:"type"`
	Priority               *string `json:"priority"`
	Description            *string `json:"description"`
	ClientOrRequestingUnit *string `json:"client_or_requesting_unit"`
	ScopeOrObjective       *string `json:"scope_or_objective"`
}

type createProjectResponse struct {
	Project projectResponse `json:"project"`
	Member  memberResponse  `json:"member"`
}

type projectListItemResponse struct {
	Project projectResponse `json:"project"`
	Member  memberResponse  `json:"member"`
}

type projectResponse struct {
	ID                     string `json:"id"`
	WorkspaceID            string `json:"workspace_id"`
	Code                   string `json:"code"`
	Name                   string `json:"name"`
	Type                   string `json:"type"`
	Status                 string `json:"status"`
	Priority               string `json:"priority"`
	Description            string `json:"description,omitempty"`
	ClientOrRequestingUnit string `json:"client_or_requesting_unit,omitempty"`
	ScopeOrObjective       string `json:"scope_or_objective,omitempty"`
}

type memberResponse struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Role      string `json:"role"`
	Status    string `json:"status"`
}

func NewHandler(projectService ProjectService, sessionService SessionService, tenantResolver TenantResolver, cookie CookieConfig) Handler {
	return Handler{
		projects: projectService,
		session:  sessionService,
		tenant:   tenantResolver,
		cookie:   cookie,
	}
}

func (h Handler) RegisterRoutes(router fiber.Router) {
	projects := router.Group("/workspace/projects")
	projects.Get("/", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceView), h.ListProjects)
	projects.Post("/", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceManage), h.CreateProject)
	projects.Get("/:project_id", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceView), h.GetProject)
	projects.Patch("/:project_id", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceManage), h.UpdateProject)
}

func (h Handler) CreateProject(c fiber.Ctx) error {
	if h.projects == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}

	var req createProjectRequest
	if err := c.Bind().Body(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Request body is invalid")
	}

	result, err := h.projects.CreateProject(c.Context(), projectsvc.CreateProjectInput{
		Account:                account,
		TenantContext:          tenantContext,
		Name:                   req.Name,
		Type:                   project.ProjectType(req.Type),
		Description:            req.Description,
		ClientOrRequestingUnit: req.ClientOrRequestingUnit,
		ScopeOrObjective:       req.ScopeOrObjective,
	})
	if err != nil {
		return renderProjectError(c, err)
	}

	return presenter.RenderItem(c, createProjectResponse{
		Project: toProjectResponse(result.Project),
		Member:  toMemberResponse(result.Member),
	}, fiber.StatusCreated)
}

func (h Handler) ListProjects(c fiber.Ctx) error {
	if h.projects == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}

	query := presenter.ParseOffsetQuery(c)
	result, err := h.projects.ListProjects(c.Context(), projectsvc.ListProjectsInput{
		Account:       account,
		TenantContext: tenantContext,
		Limit:         query.Limit,
		Offset:        query.Offset,
	})
	if err != nil {
		return renderProjectError(c, err)
	}

	items := make([]projectListItemResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, projectListItemResponse{
			Project: toProjectResponse(item.Project),
			Member:  toMemberResponse(item.Member),
		})
	}

	return presenter.RenderList(c, items, presenter.NewOffsetPagination(result.Total, query.Limit, query.Offset))
}

func (h Handler) GetProject(c fiber.Ctx) error {
	if h.projects == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}

	projectID, err := uuid.Parse(c.Params("project_id"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "PROJECT_ID_INVALID", "Project id is invalid")
	}

	result, err := h.projects.GetProject(c.Context(), projectsvc.GetProjectInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
	})
	if err != nil {
		return renderProjectError(c, err)
	}

	return presenter.RenderItem(c, projectListItemResponse{
		Project: toProjectResponse(result.Item.Project),
		Member:  toMemberResponse(result.Item.Member),
	})
}

func (h Handler) UpdateProject(c fiber.Ctx) error {
	if h.projects == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}

	projectID, err := uuid.Parse(c.Params("project_id"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "PROJECT_ID_INVALID", "Project id is invalid")
	}

	var req updateProjectRequest
	if err := c.Bind().Body(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Request body is invalid")
	}

	var projectType *project.ProjectType
	if req.Type != nil {
		value := project.ProjectType(*req.Type)
		projectType = &value
	}
	var priority *project.ProjectPriority
	if req.Priority != nil {
		value := project.ProjectPriority(*req.Priority)
		priority = &value
	}

	result, err := h.projects.UpdateProject(c.Context(), projectsvc.UpdateProjectInput{
		Account:                account,
		TenantContext:          tenantContext,
		ProjectID:              projectID,
		Name:                   req.Name,
		Type:                   projectType,
		Priority:               priority,
		Description:            req.Description,
		ClientOrRequestingUnit: req.ClientOrRequestingUnit,
		ScopeOrObjective:       req.ScopeOrObjective,
	})
	if err != nil {
		return renderProjectError(c, err)
	}

	return presenter.RenderItem(c, projectListItemResponse{
		Project: toProjectResponse(result.Item.Project),
		Member:  toMemberResponse(result.Item.Member),
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
	if h.tenant == nil {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}

	account, ok := c.Locals(accountLocalKey).(auth.UserAccount)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}

	result, err := h.tenant.ResolveTenantContext(c.Context(), workspacesvc.ResolveTenantContextInput{
		Account:       account,
		WorkspaceSlug: c.Get(workspaceSlugHeader),
	})
	if err != nil {
		return renderWorkspaceError(c, err)
	}

	c.Locals(tenantContextLocalKey, result.Context)
	return c.Next()
}

func (h Handler) requireWorkspacePermission(permission workspaceperm.Permission) fiber.Handler {
	return func(c fiber.Ctx) error {
		context, ok := c.Locals(tenantContextLocalKey).(workspace.TenantContext)
		if !ok {
			return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
		}
		if !workspaceperm.Can(context.Role, permission) {
			return presenter.RenderError(c, fiber.StatusForbidden, "PERMISSION_DENIED", "Permission denied")
		}
		return c.Next()
	}
}

func (h Handler) accountAndTenantContext(c fiber.Ctx) (auth.UserAccount, workspace.TenantContext, bool) {
	account, accountOK := c.Locals(accountLocalKey).(auth.UserAccount)
	tenantContext, tenantOK := c.Locals(tenantContextLocalKey).(workspace.TenantContext)
	return account, tenantContext, accountOK && tenantOK
}

func (h Handler) NotImplemented(c fiber.Ctx) error {
	return presenter.RenderError(c, fiber.StatusNotImplemented, "NOT_IMPLEMENTED", "Project endpoint is not implemented yet")
}

func (h Handler) cookieName() string {
	if h.cookie.Name == "" {
		return "prasankit_session"
	}
	return h.cookie.Name
}

func renderProjectError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, projectsvc.ErrProjectNameRequired):
		return presenter.RenderError(c, fiber.StatusBadRequest, "PROJECT_NAME_REQUIRED", "Project name is required")
	case errors.Is(err, projectsvc.ErrProjectTypeInvalid):
		return presenter.RenderError(c, fiber.StatusBadRequest, "PROJECT_TYPE_INVALID", "Project type is invalid")
	case errors.Is(err, projectsvc.ErrProjectPriorityInvalid):
		return presenter.RenderError(c, fiber.StatusBadRequest, "PROJECT_PRIORITY_INVALID", "Project priority is invalid")
	case errors.Is(err, projectsvc.ErrProjectUpdateNoFields):
		return presenter.RenderError(c, fiber.StatusBadRequest, "PROJECT_UPDATE_NO_FIELDS", "Project update has no fields")
	case errors.Is(err, projectsvc.ErrAccountInactive):
		return presenter.RenderError(c, fiber.StatusForbidden, "ACCOUNT_INACTIVE", "Account is inactive")
	case errors.Is(err, projectsvc.ErrTenantContextRequired):
		return presenter.RenderError(c, fiber.StatusForbidden, "TENANT_CONTEXT_REQUIRED", "Tenant context is required")
	case errors.Is(err, projectsvc.ErrProjectIDRequired):
		return presenter.RenderError(c, fiber.StatusBadRequest, "PROJECT_ID_REQUIRED", "Project id is required")
	case errors.Is(err, projectsvc.ErrProjectNotFound):
		return presenter.RenderError(c, fiber.StatusNotFound, "PROJECT_NOT_FOUND", "Project not found")
	default:
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
}

func renderWorkspaceError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, workspacesvc.ErrWorkspaceSlugRequired):
		return presenter.RenderError(c, fiber.StatusBadRequest, "WORKSPACE_SLUG_REQUIRED", "Workspace slug is required")
	case errors.Is(err, workspacesvc.ErrWorkspaceSlugInvalid):
		return presenter.RenderError(c, fiber.StatusBadRequest, "WORKSPACE_SLUG_INVALID", "Workspace slug is invalid")
	case errors.Is(err, workspacesvc.ErrWorkspaceAccessDenied):
		return presenter.RenderError(c, fiber.StatusForbidden, "WORKSPACE_ACCESS_DENIED", "Workspace access denied")
	case errors.Is(err, workspacesvc.ErrAccountInactive):
		return presenter.RenderError(c, fiber.StatusForbidden, "ACCOUNT_INACTIVE", "Account is inactive")
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

func toProjectResponse(value project.Project) projectResponse {
	return projectResponse{
		ID:                     value.ID.String(),
		WorkspaceID:            value.WorkspaceID.String(),
		Code:                   value.Code,
		Name:                   value.Name,
		Type:                   string(value.Type),
		Status:                 string(value.Status),
		Priority:               string(value.Priority),
		Description:            value.Description,
		ClientOrRequestingUnit: value.ClientOrRequestingUnit,
		ScopeOrObjective:       value.ScopeOrObjective,
	}
}

func toMemberResponse(value project.Member) memberResponse {
	return memberResponse{
		ID:        value.ID.String(),
		ProjectID: value.ProjectID.String(),
		Role:      string(value.Role),
		Status:    string(value.Status),
	}
}
