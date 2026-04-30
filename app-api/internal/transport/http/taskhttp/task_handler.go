package taskhttp

import (
	"context"
	"errors"
	"time"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/auth/authsvc"
	"prasankit-api/internal/modules/task"
	"prasankit-api/internal/modules/task/tasksvc"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/internal/modules/workspace/workspaceperm"
	"prasankit-api/internal/modules/workspace/workspacesvc"
	"prasankit-api/internal/transport/http/presenter"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const accountLocalKey = "task.account"
const tenantContextLocalKey = "task.tenant_context"
const workspaceSlugHeader = "X-Workspace-Slug"

type TaskService interface {
	CreateTask(ctx context.Context, input tasksvc.CreateTaskInput) (*tasksvc.CreateTaskResult, error)
	ListTasks(ctx context.Context, input tasksvc.ListTasksInput) (*tasksvc.ListTasksResult, error)
	GetTask(ctx context.Context, input tasksvc.GetTaskInput) (*tasksvc.GetTaskResult, error)
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
	tasks   TaskService
	session SessionService
	tenant  TenantResolver
	cookie  CookieConfig
}

type createTaskRequest struct {
	Title            string  `json:"title"`
	Priority         string  `json:"priority"`
	AssigneeMemberID *string `json:"assignee_member_id"`
	Description      string  `json:"description"`
	StartDate        *string `json:"start_date"`
	DueDate          *string `json:"due_date"`
}

type taskResponse struct {
	ID               string  `json:"id"`
	ProjectID        string  `json:"project_id"`
	No               string  `json:"no"`
	Title            string  `json:"title"`
	Status           string  `json:"status"`
	Priority         string  `json:"priority"`
	AssigneeMemberID *string `json:"assignee_member_id,omitempty"`
	Description      string  `json:"description,omitempty"`
	StartDate        *string `json:"start_date,omitempty"`
	DueDate          *string `json:"due_date,omitempty"`
	CompletedDate    *string `json:"completed_date,omitempty"`
}

func NewHandler(taskService TaskService, sessionService SessionService, tenantResolver TenantResolver, cookie CookieConfig) Handler {
	return Handler{
		tasks:   taskService,
		session: sessionService,
		tenant:  tenantResolver,
		cookie:  cookie,
	}
}

func (h Handler) RegisterRoutes(router fiber.Router) {
	tasks := router.Group("/workspace/projects/:project_id/tasks")
	tasks.Get("/", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceView), h.ListTasks)
	tasks.Post("/", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceManage), h.CreateTask)
	tasks.Get("/:task_id", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceView), h.GetTask)
}

func (h Handler) CreateTask(c fiber.Ctx) error {
	if h.tasks == nil {
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

	var req createTaskRequest
	if err := c.Bind().Body(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Request body is invalid")
	}

	assigneeMemberID, err := parseOptionalUUID(req.AssigneeMemberID)
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_ASSIGNEE_MEMBER_ID_INVALID", "Task assignee member id is invalid")
	}
	startDate, err := parseOptionalDate(req.StartDate)
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_START_DATE_INVALID", "Task start date is invalid")
	}
	dueDate, err := parseOptionalDate(req.DueDate)
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_DUE_DATE_INVALID", "Task due date is invalid")
	}

	result, err := h.tasks.CreateTask(c.Context(), tasksvc.CreateTaskInput{
		Account:          account,
		TenantContext:    tenantContext,
		ProjectID:        projectID,
		Title:            req.Title,
		Priority:         task.Priority(req.Priority),
		AssigneeMemberID: assigneeMemberID,
		Description:      req.Description,
		StartDate:        startDate,
		DueDate:          dueDate,
	})
	if err != nil {
		return renderTaskError(c, err)
	}

	return presenter.RenderItem(c, toTaskResponse(result.Task), fiber.StatusCreated)
}

func (h Handler) ListTasks(c fiber.Ctx) error {
	if h.tasks == nil {
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

	query := presenter.ParseOffsetQuery(c)
	result, err := h.tasks.ListTasks(c.Context(), tasksvc.ListTasksInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		Limit:         query.Limit,
		Offset:        query.Offset,
	})
	if err != nil {
		return renderTaskError(c, err)
	}

	items := make([]taskResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toTaskResponse(item))
	}
	return presenter.RenderList(c, items, presenter.NewOffsetPagination(result.Total, query.Limit, query.Offset))
}

func (h Handler) GetTask(c fiber.Ctx) error {
	if h.tasks == nil {
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
	taskID, err := uuid.Parse(c.Params("task_id"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_ID_INVALID", "Task id is invalid")
	}

	result, err := h.tasks.GetTask(c.Context(), tasksvc.GetTaskInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
	})
	if err != nil {
		return renderTaskError(c, err)
	}

	return presenter.RenderItem(c, toTaskResponse(result.Task))
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
	return presenter.RenderError(c, fiber.StatusNotImplemented, "NOT_IMPLEMENTED", "Task endpoint is not implemented yet")
}

func (h Handler) cookieName() string {
	if h.cookie.Name == "" {
		return "prasankit_session"
	}
	return h.cookie.Name
}

func renderTaskError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, tasksvc.ErrTaskTitleRequired):
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_TITLE_REQUIRED", "Task title is required")
	case errors.Is(err, tasksvc.ErrTaskPriorityInvalid):
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_PRIORITY_INVALID", "Task priority is invalid")
	case errors.Is(err, tasksvc.ErrTaskAssigneeNotFound):
		return presenter.RenderError(c, fiber.StatusNotFound, "TASK_ASSIGNEE_NOT_FOUND", "Task assignee not found")
	case errors.Is(err, tasksvc.ErrTaskIDRequired):
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_ID_REQUIRED", "Task id is required")
	case errors.Is(err, tasksvc.ErrTaskNotFound):
		return presenter.RenderError(c, fiber.StatusNotFound, "TASK_NOT_FOUND", "Task not found")
	case errors.Is(err, tasksvc.ErrAccountInactive):
		return presenter.RenderError(c, fiber.StatusForbidden, "ACCOUNT_INACTIVE", "Account is inactive")
	case errors.Is(err, tasksvc.ErrTenantContextRequired):
		return presenter.RenderError(c, fiber.StatusForbidden, "TENANT_CONTEXT_REQUIRED", "Tenant context is required")
	case errors.Is(err, tasksvc.ErrProjectIDRequired):
		return presenter.RenderError(c, fiber.StatusBadRequest, "PROJECT_ID_REQUIRED", "Project id is required")
	case errors.Is(err, tasksvc.ErrProjectNotFound):
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

func parseOptionalUUID(value *string) (*uuid.UUID, error) {
	if value == nil || *value == "" {
		return nil, nil
	}
	id, err := uuid.Parse(*value)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func parseOptionalDate(value *string) (*time.Time, error) {
	if value == nil || *value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.DateOnly, *value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func toTaskResponse(value task.Task) taskResponse {
	var assigneeMemberID *string
	if value.AssigneeMemberID != nil {
		id := value.AssigneeMemberID.String()
		assigneeMemberID = &id
	}

	return taskResponse{
		ID:               value.ID.String(),
		ProjectID:        value.ProjectID.String(),
		No:               value.No,
		Title:            value.Title,
		Status:           string(value.Status),
		Priority:         string(value.Priority),
		AssigneeMemberID: assigneeMemberID,
		Description:      value.Description,
		StartDate:        formatDate(value.StartDate),
		DueDate:          formatDate(value.DueDate),
		CompletedDate:    formatDate(value.CompletedDate),
	}
}

func formatDate(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format(time.DateOnly)
	return &formatted
}
