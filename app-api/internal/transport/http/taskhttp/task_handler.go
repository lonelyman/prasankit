package taskhttp

import (
	"context"
	"errors"
	"strings"
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
	GetTaskBoardSummary(ctx context.Context, input tasksvc.GetTaskBoardSummaryInput) (*tasksvc.GetTaskBoardSummaryResult, error)
	GetTask(ctx context.Context, input tasksvc.GetTaskInput) (*tasksvc.GetTaskResult, error)
	ListTaskActivities(ctx context.Context, input tasksvc.ListTaskActivitiesInput) (*tasksvc.ListTaskActivitiesResult, error)
	CreateTaskComment(ctx context.Context, input tasksvc.CreateTaskCommentInput) (*tasksvc.CreateTaskCommentResult, error)
	ListTaskComments(ctx context.Context, input tasksvc.ListTaskCommentsInput) (*tasksvc.ListTaskCommentsResult, error)
	DeleteTaskComment(ctx context.Context, input tasksvc.DeleteTaskCommentInput) error
	CreateTaskChecklistItem(ctx context.Context, input tasksvc.CreateTaskChecklistItemInput) (*tasksvc.CreateTaskChecklistItemResult, error)
	ListTaskChecklistItems(ctx context.Context, input tasksvc.ListTaskChecklistItemsInput) (*tasksvc.ListTaskChecklistItemsResult, error)
	UpdateTaskChecklistItem(ctx context.Context, input tasksvc.UpdateTaskChecklistItemInput) (*tasksvc.UpdateTaskChecklistItemResult, error)
	DeleteTaskChecklistItem(ctx context.Context, input tasksvc.DeleteTaskChecklistItemInput) error
	CreateTaskAttachmentUpload(ctx context.Context, input tasksvc.CreateTaskAttachmentUploadInput) (*tasksvc.CreateTaskAttachmentUploadResult, error)
	CompleteTaskAttachmentUpload(ctx context.Context, input tasksvc.CompleteTaskAttachmentUploadInput) error
	GetTaskAttachmentDownloadURL(ctx context.Context, input tasksvc.GetTaskAttachmentDownloadURLInput) (*tasksvc.GetTaskAttachmentDownloadURLResult, error)
	ListTaskAttachments(ctx context.Context, input tasksvc.ListTaskAttachmentsInput) (*tasksvc.ListTaskAttachmentsResult, error)
	DeleteTaskAttachment(ctx context.Context, input tasksvc.DeleteTaskAttachmentInput) error
	AssignTaskTag(ctx context.Context, input tasksvc.AssignTaskTagInput) (*tasksvc.AssignTaskTagResult, error)
	ListTaskTags(ctx context.Context, input tasksvc.ListTaskTagsInput) (*tasksvc.ListTaskTagsResult, error)
	RemoveTaskTag(ctx context.Context, input tasksvc.RemoveTaskTagInput) error
	CreateTaskRelation(ctx context.Context, input tasksvc.CreateTaskRelationInput) (*tasksvc.CreateTaskRelationResult, error)
	ListTaskRelations(ctx context.Context, input tasksvc.ListTaskRelationsInput) (*tasksvc.ListTaskRelationsResult, error)
	DeleteTaskRelation(ctx context.Context, input tasksvc.DeleteTaskRelationInput) error
	UpdateTask(ctx context.Context, input tasksvc.UpdateTaskInput) (*tasksvc.UpdateTaskResult, error)
	UpdateTaskStatus(ctx context.Context, input tasksvc.UpdateTaskStatusInput) (*tasksvc.UpdateTaskStatusResult, error)
	DeleteTask(ctx context.Context, input tasksvc.DeleteTaskInput) error
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

type updateTaskRequest struct {
	Title            *string `json:"title"`
	Priority         *string `json:"priority"`
	AssigneeMemberID *string `json:"assignee_member_id"`
	Description      *string `json:"description"`
	StartDate        *string `json:"start_date"`
	DueDate          *string `json:"due_date"`
}

type updateTaskStatusRequest struct {
	Status string `json:"status"`
}

type createTaskCommentRequest struct {
	Body string `json:"body"`
}

type createTaskChecklistItemRequest struct {
	Text      string `json:"text"`
	SortOrder int    `json:"sort_order"`
}

type updateTaskChecklistItemRequest struct {
	Text        *string `json:"text"`
	IsCompleted *bool   `json:"is_completed"`
	SortOrder   *int    `json:"sort_order"`
}

type createTaskAttachmentUploadRequest struct {
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

type assignTaskTagRequest struct {
	Name  string  `json:"name"`
	Color *string `json:"color"`
}

type createTaskRelationRequest struct {
	TargetTaskID string `json:"target_task_id"`
	Type         string `json:"type"`
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

type taskBoardSummaryResponse struct {
	ProjectID string         `json:"project_id"`
	Total     int            `json:"total"`
	Counts    map[string]int `json:"counts"`
}

type taskActivityResponse struct {
	ID             string         `json:"id"`
	TaskID         string         `json:"task_id"`
	ActorAccountID string         `json:"actor_account_id"`
	Action         string         `json:"action"`
	FromStatus     *string        `json:"from_status,omitempty"`
	ToStatus       *string        `json:"to_status,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	CreatedAt      string         `json:"created_at"`
}

type taskCommentResponse struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	Body      string `json:"body"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type taskChecklistItemResponse struct {
	ID          string  `json:"id"`
	TaskID      string  `json:"task_id"`
	Text        string  `json:"text"`
	IsCompleted bool    `json:"is_completed"`
	SortOrder   int     `json:"sort_order"`
	CreatedBy   string  `json:"created_by"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
	CompletedBy *string `json:"completed_by,omitempty"`
	CompletedAt *string `json:"completed_at,omitempty"`
}

type taskAttachmentResponse struct {
	ID          string  `json:"id"`
	TaskID      string  `json:"task_id"`
	FileName    string  `json:"file_name"`
	ContentType string  `json:"content_type"`
	SizeBytes   int64   `json:"size_bytes"`
	Status      string  `json:"status"`
	UploadedBy  string  `json:"uploaded_by"`
	UploadedAt  *string `json:"uploaded_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

type taskAttachmentUploadResponse struct {
	Attachment taskAttachmentResponse `json:"attachment"`
	UploadURL  string                 `json:"upload_url"`
	ExpiresAt  string                 `json:"expires_at"`
}

type taskAttachmentDownloadResponse struct {
	Attachment  taskAttachmentResponse `json:"attachment"`
	DownloadURL string                 `json:"download_url"`
	ExpiresAt   string                 `json:"expires_at"`
}

type taskTagResponse struct {
	ID        string  `json:"id"`
	ProjectID string  `json:"project_id"`
	Name      string  `json:"name"`
	Color     *string `json:"color,omitempty"`
	CreatedBy string  `json:"created_by"`
	CreatedAt string  `json:"created_at"`
}

type taskRelationResponse struct {
	ID           string `json:"id"`
	ProjectID    string `json:"project_id"`
	SourceTaskID string `json:"source_task_id"`
	TargetTaskID string `json:"target_task_id"`
	Type         string `json:"type"`
	CreatedBy    string `json:"created_by"`
	CreatedAt    string `json:"created_at"`
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
	tasks.Get("/summary", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceView), h.GetTaskBoardSummary)
	tasks.Get("/:task_id/activities", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceView), h.ListTaskActivities)
	tasks.Get("/:task_id/comments", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceView), h.ListTaskComments)
	tasks.Post("/:task_id/comments", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceManage), h.CreateTaskComment)
	tasks.Delete("/:task_id/comments/:comment_id", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceManage), h.DeleteTaskComment)
	tasks.Get("/:task_id/checklist", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceView), h.ListTaskChecklistItems)
	tasks.Post("/:task_id/checklist", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceManage), h.CreateTaskChecklistItem)
	tasks.Patch("/:task_id/checklist/:item_id", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceManage), h.UpdateTaskChecklistItem)
	tasks.Delete("/:task_id/checklist/:item_id", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceManage), h.DeleteTaskChecklistItem)
	tasks.Get("/:task_id/attachments", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceView), h.ListTaskAttachments)
	tasks.Post("/:task_id/attachments/uploads", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceManage), h.CreateTaskAttachmentUpload)
	tasks.Patch("/:task_id/attachments/:attachment_id/complete", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceManage), h.CompleteTaskAttachmentUpload)
	tasks.Get("/:task_id/attachments/:attachment_id/download", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceView), h.GetTaskAttachmentDownloadURL)
	tasks.Delete("/:task_id/attachments/:attachment_id", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceManage), h.DeleteTaskAttachment)
	tasks.Get("/:task_id/tags", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceView), h.ListTaskTags)
	tasks.Post("/:task_id/tags", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceManage), h.AssignTaskTag)
	tasks.Delete("/:task_id/tags/:tag_id", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceManage), h.RemoveTaskTag)
	tasks.Get("/:task_id/relations", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceView), h.ListTaskRelations)
	tasks.Post("/:task_id/relations", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceManage), h.CreateTaskRelation)
	tasks.Delete("/:task_id/relations/:relation_id", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceManage), h.DeleteTaskRelation)
	tasks.Get("/:task_id", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceView), h.GetTask)
	tasks.Patch("/:task_id/status", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceManage), h.UpdateTaskStatus)
	tasks.Patch("/:task_id", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceManage), h.UpdateTask)
	tasks.Delete("/:task_id", h.requireSession, h.requireTenantContext, h.requireWorkspacePermission(workspaceperm.PermissionWorkspaceManage), h.DeleteTask)
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
	status := parseTaskStatusQuery(c.Query("status"))
	priority := parseTaskPriorityQuery(c.Query("priority"))
	assigneeMemberID, err := parseTaskAssigneeQuery(c.Query("assignee_member_id"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_ASSIGNEE_MEMBER_ID_INVALID", "Task assignee member id is invalid")
	}
	tagID, err := parseTaskTagQuery(c.Query("tag_id"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_TAG_ID_INVALID", "Task tag id is invalid")
	}
	result, err := h.tasks.ListTasks(c.Context(), tasksvc.ListTasksInput{
		Account:          account,
		TenantContext:    tenantContext,
		ProjectID:        projectID,
		Status:           status,
		Priority:         priority,
		AssigneeMemberID: assigneeMemberID,
		TagID:            tagID,
		Limit:            query.Limit,
		Offset:           query.Offset,
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

func (h Handler) GetTaskBoardSummary(c fiber.Ctx) error {
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

	result, err := h.tasks.GetTaskBoardSummary(c.Context(), tasksvc.GetTaskBoardSummaryInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
	})
	if err != nil {
		return renderTaskError(c, err)
	}

	return presenter.RenderItem(c, toTaskBoardSummaryResponse(*result))
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

func (h Handler) ListTaskActivities(c fiber.Ctx) error {
	if h.tasks == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}

	projectID, taskID, err := parseProjectAndTaskIDs(c)
	if err != nil {
		return err
	}

	query := presenter.ParseOffsetQuery(c)
	result, err := h.tasks.ListTaskActivities(c.Context(), tasksvc.ListTaskActivitiesInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		Limit:         query.Limit,
		Offset:        query.Offset,
	})
	if err != nil {
		return renderTaskError(c, err)
	}

	items := make([]taskActivityResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toTaskActivityResponse(item))
	}
	return presenter.RenderList(c, items, presenter.NewOffsetPagination(result.Total, query.Limit, query.Offset))
}

func (h Handler) CreateTaskComment(c fiber.Ctx) error {
	if h.tasks == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
	projectID, taskID, err := parseProjectAndTaskIDs(c)
	if err != nil {
		return err
	}
	var req createTaskCommentRequest
	if err := c.Bind().Body(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Request body is invalid")
	}

	result, err := h.tasks.CreateTaskComment(c.Context(), tasksvc.CreateTaskCommentInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		Body:          req.Body,
	})
	if err != nil {
		return renderTaskError(c, err)
	}

	return presenter.RenderItem(c, toTaskCommentResponse(result.Comment), fiber.StatusCreated)
}

func (h Handler) ListTaskComments(c fiber.Ctx) error {
	if h.tasks == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
	projectID, taskID, err := parseProjectAndTaskIDs(c)
	if err != nil {
		return err
	}

	query := presenter.ParseOffsetQuery(c)
	result, err := h.tasks.ListTaskComments(c.Context(), tasksvc.ListTaskCommentsInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		Limit:         query.Limit,
		Offset:        query.Offset,
	})
	if err != nil {
		return renderTaskError(c, err)
	}

	items := make([]taskCommentResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toTaskCommentResponse(item))
	}
	return presenter.RenderList(c, items, presenter.NewOffsetPagination(result.Total, query.Limit, query.Offset))
}

func (h Handler) DeleteTaskComment(c fiber.Ctx) error {
	if h.tasks == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
	projectID, taskID, err := parseProjectAndTaskIDs(c)
	if err != nil {
		return err
	}
	commentID, err := uuid.Parse(c.Params("comment_id"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_COMMENT_ID_INVALID", "Task comment id is invalid")
	}

	if err := h.tasks.DeleteTaskComment(c.Context(), tasksvc.DeleteTaskCommentInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		CommentID:     commentID,
	}); err != nil {
		return renderTaskError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h Handler) CreateTaskChecklistItem(c fiber.Ctx) error {
	if h.tasks == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
	projectID, taskID, err := parseProjectAndTaskIDs(c)
	if err != nil {
		return err
	}
	var req createTaskChecklistItemRequest
	if err := c.Bind().Body(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Request body is invalid")
	}

	result, err := h.tasks.CreateTaskChecklistItem(c.Context(), tasksvc.CreateTaskChecklistItemInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		Text:          req.Text,
		SortOrder:     req.SortOrder,
	})
	if err != nil {
		return renderTaskError(c, err)
	}

	return presenter.RenderItem(c, toTaskChecklistItemResponse(result.Item), fiber.StatusCreated)
}

func (h Handler) ListTaskChecklistItems(c fiber.Ctx) error {
	if h.tasks == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
	projectID, taskID, err := parseProjectAndTaskIDs(c)
	if err != nil {
		return err
	}

	result, err := h.tasks.ListTaskChecklistItems(c.Context(), tasksvc.ListTaskChecklistItemsInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
	})
	if err != nil {
		return renderTaskError(c, err)
	}

	items := make([]taskChecklistItemResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toTaskChecklistItemResponse(item))
	}
	return presenter.RenderItem(c, map[string]any{"items": items})
}

func (h Handler) UpdateTaskChecklistItem(c fiber.Ctx) error {
	if h.tasks == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
	projectID, taskID, err := parseProjectAndTaskIDs(c)
	if err != nil {
		return err
	}
	itemID, err := uuid.Parse(c.Params("item_id"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_CHECKLIST_ITEM_ID_INVALID", "Task checklist item id is invalid")
	}
	var req updateTaskChecklistItemRequest
	if err := c.Bind().Body(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Request body is invalid")
	}

	result, err := h.tasks.UpdateTaskChecklistItem(c.Context(), tasksvc.UpdateTaskChecklistItemInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		ItemID:        itemID,
		Text:          req.Text,
		IsCompleted:   req.IsCompleted,
		SortOrder:     req.SortOrder,
	})
	if err != nil {
		return renderTaskError(c, err)
	}

	return presenter.RenderItem(c, toTaskChecklistItemResponse(result.Item))
}

func (h Handler) DeleteTaskChecklistItem(c fiber.Ctx) error {
	if h.tasks == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
	projectID, taskID, err := parseProjectAndTaskIDs(c)
	if err != nil {
		return err
	}
	itemID, err := uuid.Parse(c.Params("item_id"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_CHECKLIST_ITEM_ID_INVALID", "Task checklist item id is invalid")
	}

	if err := h.tasks.DeleteTaskChecklistItem(c.Context(), tasksvc.DeleteTaskChecklistItemInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		ItemID:        itemID,
	}); err != nil {
		return renderTaskError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h Handler) CreateTaskAttachmentUpload(c fiber.Ctx) error {
	if h.tasks == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
	projectID, taskID, err := parseProjectAndTaskIDs(c)
	if err != nil {
		return err
	}
	var req createTaskAttachmentUploadRequest
	if err := c.Bind().Body(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Request body is invalid")
	}

	result, err := h.tasks.CreateTaskAttachmentUpload(c.Context(), tasksvc.CreateTaskAttachmentUploadInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		FileName:      req.FileName,
		ContentType:   req.ContentType,
		SizeBytes:     req.SizeBytes,
	})
	if err != nil {
		return renderTaskError(c, err)
	}

	return presenter.RenderItem(c, taskAttachmentUploadResponse{
		Attachment: toTaskAttachmentResponse(result.Attachment),
		UploadURL:  result.UploadURL.URL,
		ExpiresAt:  result.UploadURL.ExpiresAt.Format(time.RFC3339),
	}, fiber.StatusCreated)
}

func (h Handler) CompleteTaskAttachmentUpload(c fiber.Ctx) error {
	if h.tasks == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
	projectID, taskID, err := parseProjectAndTaskIDs(c)
	if err != nil {
		return err
	}
	attachmentID, err := uuid.Parse(c.Params("attachment_id"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_ATTACHMENT_ID_INVALID", "Task attachment id is invalid")
	}

	if err := h.tasks.CompleteTaskAttachmentUpload(c.Context(), tasksvc.CompleteTaskAttachmentUploadInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		AttachmentID:  attachmentID,
	}); err != nil {
		return renderTaskError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h Handler) GetTaskAttachmentDownloadURL(c fiber.Ctx) error {
	if h.tasks == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
	projectID, taskID, err := parseProjectAndTaskIDs(c)
	if err != nil {
		return err
	}
	attachmentID, err := uuid.Parse(c.Params("attachment_id"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_ATTACHMENT_ID_INVALID", "Task attachment id is invalid")
	}

	result, err := h.tasks.GetTaskAttachmentDownloadURL(c.Context(), tasksvc.GetTaskAttachmentDownloadURLInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		AttachmentID:  attachmentID,
	})
	if err != nil {
		return renderTaskError(c, err)
	}

	return presenter.RenderItem(c, taskAttachmentDownloadResponse{
		Attachment:  toTaskAttachmentResponse(result.Attachment),
		DownloadURL: result.DownloadURL.URL,
		ExpiresAt:   result.DownloadURL.ExpiresAt.Format(time.RFC3339),
	})
}

func (h Handler) ListTaskAttachments(c fiber.Ctx) error {
	if h.tasks == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
	projectID, taskID, err := parseProjectAndTaskIDs(c)
	if err != nil {
		return err
	}

	query := presenter.ParseOffsetQuery(c)
	result, err := h.tasks.ListTaskAttachments(c.Context(), tasksvc.ListTaskAttachmentsInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		Limit:         query.Limit,
		Offset:        query.Offset,
	})
	if err != nil {
		return renderTaskError(c, err)
	}

	items := make([]taskAttachmentResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toTaskAttachmentResponse(item))
	}
	return presenter.RenderList(c, items, presenter.NewOffsetPagination(result.Total, query.Limit, query.Offset))
}

func (h Handler) DeleteTaskAttachment(c fiber.Ctx) error {
	if h.tasks == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
	projectID, taskID, err := parseProjectAndTaskIDs(c)
	if err != nil {
		return err
	}
	attachmentID, err := uuid.Parse(c.Params("attachment_id"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_ATTACHMENT_ID_INVALID", "Task attachment id is invalid")
	}

	if err := h.tasks.DeleteTaskAttachment(c.Context(), tasksvc.DeleteTaskAttachmentInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		AttachmentID:  attachmentID,
	}); err != nil {
		return renderTaskError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h Handler) AssignTaskTag(c fiber.Ctx) error {
	if h.tasks == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
	projectID, taskID, err := parseProjectAndTaskIDs(c)
	if err != nil {
		return err
	}
	var req assignTaskTagRequest
	if err := c.Bind().Body(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Request body is invalid")
	}

	result, err := h.tasks.AssignTaskTag(c.Context(), tasksvc.AssignTaskTagInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		Name:          req.Name,
		Color:         req.Color,
	})
	if err != nil {
		return renderTaskError(c, err)
	}

	return presenter.RenderItem(c, toTaskTagResponse(result.Tag), fiber.StatusCreated)
}

func (h Handler) ListTaskTags(c fiber.Ctx) error {
	if h.tasks == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
	projectID, taskID, err := parseProjectAndTaskIDs(c)
	if err != nil {
		return err
	}

	result, err := h.tasks.ListTaskTags(c.Context(), tasksvc.ListTaskTagsInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
	})
	if err != nil {
		return renderTaskError(c, err)
	}

	items := make([]taskTagResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toTaskTagResponse(item))
	}
	return presenter.RenderItem(c, map[string]any{"items": items})
}

func (h Handler) RemoveTaskTag(c fiber.Ctx) error {
	if h.tasks == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
	projectID, taskID, err := parseProjectAndTaskIDs(c)
	if err != nil {
		return err
	}
	tagID, err := uuid.Parse(c.Params("tag_id"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_TAG_ID_INVALID", "Task tag id is invalid")
	}

	if err := h.tasks.RemoveTaskTag(c.Context(), tasksvc.RemoveTaskTagInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		TagID:         tagID,
	}); err != nil {
		return renderTaskError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h Handler) CreateTaskRelation(c fiber.Ctx) error {
	if h.tasks == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
	projectID, taskID, err := parseProjectAndTaskIDs(c)
	if err != nil {
		return err
	}
	var req createTaskRelationRequest
	if err := c.Bind().Body(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Request body is invalid")
	}
	targetTaskID, err := uuid.Parse(req.TargetTaskID)
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_RELATION_TARGET_TASK_ID_INVALID", "Task relation target task id is invalid")
	}

	result, err := h.tasks.CreateTaskRelation(c.Context(), tasksvc.CreateTaskRelationInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		TargetTaskID:  targetTaskID,
		Type:          task.RelationType(req.Type),
	})
	if err != nil {
		return renderTaskError(c, err)
	}

	return presenter.RenderItem(c, toTaskRelationResponse(result.Relation), fiber.StatusCreated)
}

func (h Handler) ListTaskRelations(c fiber.Ctx) error {
	if h.tasks == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
	projectID, taskID, err := parseProjectAndTaskIDs(c)
	if err != nil {
		return err
	}

	result, err := h.tasks.ListTaskRelations(c.Context(), tasksvc.ListTaskRelationsInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
	})
	if err != nil {
		return renderTaskError(c, err)
	}

	items := make([]taskRelationResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toTaskRelationResponse(item))
	}
	return presenter.RenderItem(c, map[string]any{"items": items})
}

func (h Handler) DeleteTaskRelation(c fiber.Ctx) error {
	if h.tasks == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
	projectID, taskID, err := parseProjectAndTaskIDs(c)
	if err != nil {
		return err
	}
	relationID, err := uuid.Parse(c.Params("relation_id"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_RELATION_ID_INVALID", "Task relation id is invalid")
	}

	if err := h.tasks.DeleteTaskRelation(c.Context(), tasksvc.DeleteTaskRelationInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		RelationID:    relationID,
	}); err != nil {
		return renderTaskError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h Handler) UpdateTask(c fiber.Ctx) error {
	if h.tasks == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}

	projectID, taskID, err := parseProjectAndTaskIDs(c)
	if err != nil {
		return err
	}

	var req updateTaskRequest
	if err := c.Bind().Body(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Request body is invalid")
	}

	assigneeMemberID, err := parsePatchUUID(req.AssigneeMemberID)
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_ASSIGNEE_MEMBER_ID_INVALID", "Task assignee member id is invalid")
	}
	startDate, err := parsePatchDate(req.StartDate)
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_START_DATE_INVALID", "Task start date is invalid")
	}
	dueDate, err := parsePatchDate(req.DueDate)
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_DUE_DATE_INVALID", "Task due date is invalid")
	}

	var priority *task.Priority
	if req.Priority != nil {
		value := task.Priority(*req.Priority)
		priority = &value
	}

	result, err := h.tasks.UpdateTask(c.Context(), tasksvc.UpdateTaskInput{
		Account:          account,
		TenantContext:    tenantContext,
		ProjectID:        projectID,
		TaskID:           taskID,
		Title:            req.Title,
		Priority:         priority,
		AssigneeMemberID: assigneeMemberID,
		Description:      req.Description,
		StartDate:        startDate,
		DueDate:          dueDate,
	})
	if err != nil {
		return renderTaskError(c, err)
	}

	return presenter.RenderItem(c, toTaskResponse(result.Task))
}

func (h Handler) UpdateTaskStatus(c fiber.Ctx) error {
	if h.tasks == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}

	projectID, taskID, err := parseProjectAndTaskIDs(c)
	if err != nil {
		return err
	}

	var req updateTaskStatusRequest
	if err := c.Bind().Body(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Request body is invalid")
	}

	result, err := h.tasks.UpdateTaskStatus(c.Context(), tasksvc.UpdateTaskStatusInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		Status:        task.Status(req.Status),
	})
	if err != nil {
		return renderTaskError(c, err)
	}

	return presenter.RenderItem(c, toTaskResponse(result.Task))
}

func (h Handler) DeleteTask(c fiber.Ctx) error {
	if h.tasks == nil {
		return h.NotImplemented(c)
	}

	account, tenantContext, ok := h.accountAndTenantContext(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}

	projectID, taskID, err := parseProjectAndTaskIDs(c)
	if err != nil {
		return err
	}

	if err := h.tasks.DeleteTask(c.Context(), tasksvc.DeleteTaskInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
	}); err != nil {
		return renderTaskError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
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
	case errors.Is(err, tasksvc.ErrTaskUpdateNoFields):
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_UPDATE_NO_FIELDS", "Task update has no fields")
	case errors.Is(err, tasksvc.ErrTaskStatusInvalid):
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_STATUS_INVALID", "Task status is invalid")
	case errors.Is(err, tasksvc.ErrTaskPriorityInvalid):
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_PRIORITY_INVALID", "Task priority is invalid")
	case errors.Is(err, tasksvc.ErrTaskAssigneeNotFound):
		return presenter.RenderError(c, fiber.StatusNotFound, "TASK_ASSIGNEE_NOT_FOUND", "Task assignee not found")
	case errors.Is(err, tasksvc.ErrTaskCommentNotFound):
		return presenter.RenderError(c, fiber.StatusNotFound, "TASK_COMMENT_NOT_FOUND", "Task comment not found")
	case errors.Is(err, tasksvc.ErrTaskCommentBodyRequired):
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_COMMENT_BODY_REQUIRED", "Task comment body is required")
	case errors.Is(err, tasksvc.ErrTaskChecklistItemNotFound):
		return presenter.RenderError(c, fiber.StatusNotFound, "TASK_CHECKLIST_ITEM_NOT_FOUND", "Task checklist item not found")
	case errors.Is(err, tasksvc.ErrTaskChecklistTextRequired):
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_CHECKLIST_TEXT_REQUIRED", "Task checklist text is required")
	case errors.Is(err, tasksvc.ErrTaskChecklistUpdateNoFields):
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_CHECKLIST_UPDATE_NO_FIELDS", "Task checklist update has no fields")
	case errors.Is(err, tasksvc.ErrTaskChecklistSortOrderInvalid):
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_CHECKLIST_SORT_ORDER_INVALID", "Task checklist sort order is invalid")
	case errors.Is(err, tasksvc.ErrTaskAttachmentNotFound):
		return presenter.RenderError(c, fiber.StatusNotFound, "TASK_ATTACHMENT_NOT_FOUND", "Task attachment not found")
	case errors.Is(err, tasksvc.ErrAttachmentNotUploaded):
		return presenter.RenderError(c, fiber.StatusConflict, "TASK_ATTACHMENT_NOT_UPLOADED", "Task attachment is not uploaded")
	case errors.Is(err, tasksvc.ErrAttachmentFileNameRequired):
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_ATTACHMENT_FILE_NAME_REQUIRED", "Task attachment file name is required")
	case errors.Is(err, tasksvc.ErrAttachmentContentTypeRequired):
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_ATTACHMENT_CONTENT_TYPE_REQUIRED", "Task attachment content type is required")
	case errors.Is(err, tasksvc.ErrAttachmentSizeInvalid):
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_ATTACHMENT_SIZE_INVALID", "Task attachment size is invalid")
	case errors.Is(err, tasksvc.ErrAttachmentStorageNotConfigured):
		return presenter.RenderError(c, fiber.StatusInternalServerError, "TASK_ATTACHMENT_STORAGE_NOT_CONFIGURED", "Task attachment storage is not configured")
	case errors.Is(err, tasksvc.ErrTaskTagNotFound):
		return presenter.RenderError(c, fiber.StatusNotFound, "TASK_TAG_NOT_FOUND", "Task tag not found")
	case errors.Is(err, tasksvc.ErrTaskTagNameRequired):
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_TAG_NAME_REQUIRED", "Task tag name is required")
	case errors.Is(err, tasksvc.ErrTaskTagAlreadyAssigned):
		return presenter.RenderError(c, fiber.StatusConflict, "TASK_TAG_ALREADY_ASSIGNED", "Task tag is already assigned")
	case errors.Is(err, tasksvc.ErrTaskRelationNotFound):
		return presenter.RenderError(c, fiber.StatusNotFound, "TASK_RELATION_NOT_FOUND", "Task relation not found")
	case errors.Is(err, tasksvc.ErrTaskRelationTypeInvalid):
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_RELATION_TYPE_INVALID", "Task relation type is invalid")
	case errors.Is(err, tasksvc.ErrTaskRelationTargetRequired):
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_RELATION_TARGET_REQUIRED", "Task relation target task id is required")
	case errors.Is(err, tasksvc.ErrTaskRelationSelf):
		return presenter.RenderError(c, fiber.StatusBadRequest, "TASK_RELATION_SELF", "Task relation cannot target itself")
	case errors.Is(err, tasksvc.ErrTaskRelationAlreadyExists):
		return presenter.RenderError(c, fiber.StatusConflict, "TASK_RELATION_ALREADY_EXISTS", "Task relation already exists")
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

func parsePatchUUID(value *string) (**uuid.UUID, error) {
	if value == nil {
		return nil, nil
	}
	if *value == "" {
		var id *uuid.UUID
		return &id, nil
	}
	id, err := uuid.Parse(*value)
	if err != nil {
		return nil, err
	}
	idPtr := &id
	return &idPtr, nil
}

func parsePatchDate(value *string) (**time.Time, error) {
	if value == nil {
		return nil, nil
	}
	if *value == "" {
		var date *time.Time
		return &date, nil
	}
	parsed, err := time.Parse(time.DateOnly, *value)
	if err != nil {
		return nil, err
	}
	datePtr := &parsed
	return &datePtr, nil
}

func parseTaskStatusQuery(value string) *task.Status {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	status := task.Status(value)
	return &status
}

func parseTaskPriorityQuery(value string) *task.Priority {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	priority := task.Priority(value)
	return &priority
}

func parseTaskAssigneeQuery(value string) (*uuid.UUID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func parseTaskTagQuery(value string) (*uuid.UUID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func parseProjectAndTaskIDs(c fiber.Ctx) (uuid.UUID, uuid.UUID, error) {
	projectID, err := uuid.Parse(c.Params("project_id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, presenter.RenderError(c, fiber.StatusBadRequest, "PROJECT_ID_INVALID", "Project id is invalid")
	}
	taskID, err := uuid.Parse(c.Params("task_id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, presenter.RenderError(c, fiber.StatusBadRequest, "TASK_ID_INVALID", "Task id is invalid")
	}
	return projectID, taskID, nil
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

func toTaskBoardSummaryResponse(value tasksvc.GetTaskBoardSummaryResult) taskBoardSummaryResponse {
	counts := make(map[string]int, len(value.Counts))
	for _, item := range value.Counts {
		counts[string(item.Status)] = item.Count
	}
	return taskBoardSummaryResponse{
		ProjectID: value.ProjectID.String(),
		Total:     value.Total,
		Counts:    counts,
	}
}

func toTaskActivityResponse(value task.Activity) taskActivityResponse {
	return taskActivityResponse{
		ID:             value.ID.String(),
		TaskID:         value.TaskID.String(),
		ActorAccountID: value.ActorAccountID.String(),
		Action:         string(value.Action),
		FromStatus:     formatStatus(value.FromStatus),
		ToStatus:       formatStatus(value.ToStatus),
		Metadata:       value.MetadataJSON,
		CreatedAt:      value.CreatedAt.Format(time.RFC3339),
	}
}

func toTaskCommentResponse(value task.Comment) taskCommentResponse {
	return taskCommentResponse{
		ID:        value.ID.String(),
		TaskID:    value.TaskID.String(),
		Body:      value.Body,
		CreatedBy: value.CreatedBy.String(),
		CreatedAt: value.CreatedAt.Format(time.RFC3339),
		UpdatedAt: value.UpdatedAt.Format(time.RFC3339),
	}
}

func toTaskChecklistItemResponse(value task.ChecklistItem) taskChecklistItemResponse {
	return taskChecklistItemResponse{
		ID:          value.ID.String(),
		TaskID:      value.TaskID.String(),
		Text:        value.Text,
		IsCompleted: value.IsCompleted,
		SortOrder:   value.SortOrder,
		CreatedBy:   value.CreatedBy.String(),
		CreatedAt:   value.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   value.UpdatedAt.Format(time.RFC3339),
		CompletedBy: formatUUID(value.CompletedBy),
		CompletedAt: formatDateTime(value.CompletedAt),
	}
}

func toTaskAttachmentResponse(value task.Attachment) taskAttachmentResponse {
	return taskAttachmentResponse{
		ID:          value.ID.String(),
		TaskID:      value.TaskID.String(),
		FileName:    value.FileName,
		ContentType: value.ContentType,
		SizeBytes:   value.SizeBytes,
		Status:      string(value.UploadStatus),
		UploadedBy:  value.UploadedBy.String(),
		UploadedAt:  formatDateTime(value.UploadedAt),
		CreatedAt:   value.CreatedAt.Format(time.RFC3339),
	}
}

func toTaskTagResponse(value task.Tag) taskTagResponse {
	return taskTagResponse{
		ID:        value.ID.String(),
		ProjectID: value.ProjectID.String(),
		Name:      value.Name,
		Color:     value.Color,
		CreatedBy: value.CreatedBy.String(),
		CreatedAt: value.CreatedAt.Format(time.RFC3339),
	}
}

func toTaskRelationResponse(value task.Relation) taskRelationResponse {
	return taskRelationResponse{
		ID:           value.ID.String(),
		ProjectID:    value.ProjectID.String(),
		SourceTaskID: value.SourceTaskID.String(),
		TargetTaskID: value.TargetTaskID.String(),
		Type:         string(value.Type),
		CreatedBy:    value.CreatedBy.String(),
		CreatedAt:    value.CreatedAt.Format(time.RFC3339),
	}
}

func formatUUID(value *uuid.UUID) *string {
	if value == nil {
		return nil
	}
	formatted := value.String()
	return &formatted
}

func formatDateTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format(time.RFC3339)
	return &formatted
}

func formatStatus(value *task.Status) *string {
	if value == nil {
		return nil
	}
	status := string(*value)
	return &status
}

func formatDate(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format(time.DateOnly)
	return &formatted
}
