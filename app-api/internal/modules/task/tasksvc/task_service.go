package tasksvc

import (
	"context"
	"errors"
	"path"
	"strings"
	"time"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/task"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
)

const (
	defaultTaskNoPrefix = "TASK"
	defaultTaskNoLength = 4
)

var (
	ErrAccountRequired                = errors.New("account is required")
	ErrAccountInactive                = errors.New("account is inactive")
	ErrTenantContextRequired          = errors.New("tenant context is required")
	ErrProjectIDRequired              = errors.New("project id is required")
	ErrProjectNotFound                = errors.New("project not found")
	ErrTaskIDRequired                 = errors.New("task id is required")
	ErrTaskNotFound                   = errors.New("task not found")
	ErrTaskTitleRequired              = errors.New("task title is required")
	ErrTaskUpdateNoFields             = errors.New("task update has no fields")
	ErrTaskStatusInvalid              = errors.New("task status is invalid")
	ErrTaskPriorityInvalid            = errors.New("task priority is invalid")
	ErrTaskAssigneeNotFound           = errors.New("task assignee not found")
	ErrTaskCreateFail                 = errors.New("task create failed")
	ErrTaskCommentNotFound            = errors.New("task comment not found")
	ErrTaskCommentBodyRequired        = errors.New("task comment body is required")
	ErrTaskChecklistItemNotFound      = errors.New("task checklist item not found")
	ErrTaskChecklistTextRequired      = errors.New("task checklist text is required")
	ErrTaskChecklistUpdateNoFields    = errors.New("task checklist update has no fields")
	ErrTaskChecklistSortOrderInvalid  = errors.New("task checklist sort order is invalid")
	ErrTaskAttachmentNotFound         = errors.New("task attachment not found")
	ErrAttachmentNotUploaded          = errors.New("attachment is not uploaded")
	ErrAttachmentFileNameRequired     = errors.New("attachment file name is required")
	ErrAttachmentContentTypeRequired  = errors.New("attachment content type is required")
	ErrAttachmentSizeInvalid          = errors.New("attachment size is invalid")
	ErrAttachmentStorageNotConfigured = errors.New("attachment storage is not configured")
)

type CreateTaskInput struct {
	Account          auth.UserAccount
	TenantContext    workspace.TenantContext
	ProjectID        uuid.UUID
	Title            string
	Priority         task.Priority
	AssigneeMemberID *uuid.UUID
	Description      string
	StartDate        *time.Time
	DueDate          *time.Time
}

type CreateTaskResult struct {
	Task task.Task
}

type ListTasksInput struct {
	Account          auth.UserAccount
	TenantContext    workspace.TenantContext
	ProjectID        uuid.UUID
	Status           *task.Status
	Priority         *task.Priority
	AssigneeMemberID *uuid.UUID
	Limit            int
	Offset           int
}

type ListTasksResult struct {
	Items []task.Task
	Total int
}

type GetTaskBoardSummaryInput struct {
	Account       auth.UserAccount
	TenantContext workspace.TenantContext
	ProjectID     uuid.UUID
}

type GetTaskBoardSummaryResult struct {
	ProjectID uuid.UUID
	Counts    []task.StatusCount
	Total     int
}

type GetTaskInput struct {
	Account       auth.UserAccount
	TenantContext workspace.TenantContext
	ProjectID     uuid.UUID
	TaskID        uuid.UUID
}

type GetTaskResult struct {
	Task task.Task
}

type ListTaskActivitiesInput struct {
	Account       auth.UserAccount
	TenantContext workspace.TenantContext
	ProjectID     uuid.UUID
	TaskID        uuid.UUID
	Limit         int
	Offset        int
}

type ListTaskActivitiesResult struct {
	Items []task.Activity
	Total int
}

type CreateTaskCommentInput struct {
	Account       auth.UserAccount
	TenantContext workspace.TenantContext
	ProjectID     uuid.UUID
	TaskID        uuid.UUID
	Body          string
}

type CreateTaskCommentResult struct {
	Comment task.Comment
}

type ListTaskCommentsInput struct {
	Account       auth.UserAccount
	TenantContext workspace.TenantContext
	ProjectID     uuid.UUID
	TaskID        uuid.UUID
	Limit         int
	Offset        int
}

type ListTaskCommentsResult struct {
	Items []task.Comment
	Total int
}

type DeleteTaskCommentInput struct {
	Account       auth.UserAccount
	TenantContext workspace.TenantContext
	ProjectID     uuid.UUID
	TaskID        uuid.UUID
	CommentID     uuid.UUID
}

type CreateTaskChecklistItemInput struct {
	Account       auth.UserAccount
	TenantContext workspace.TenantContext
	ProjectID     uuid.UUID
	TaskID        uuid.UUID
	Text          string
	SortOrder     int
}

type CreateTaskChecklistItemResult struct {
	Item task.ChecklistItem
}

type ListTaskChecklistItemsInput struct {
	Account       auth.UserAccount
	TenantContext workspace.TenantContext
	ProjectID     uuid.UUID
	TaskID        uuid.UUID
}

type ListTaskChecklistItemsResult struct {
	Items []task.ChecklistItem
}

type UpdateTaskChecklistItemInput struct {
	Account       auth.UserAccount
	TenantContext workspace.TenantContext
	ProjectID     uuid.UUID
	TaskID        uuid.UUID
	ItemID        uuid.UUID
	Text          *string
	IsCompleted   *bool
	SortOrder     *int
}

type UpdateTaskChecklistItemResult struct {
	Item task.ChecklistItem
}

type DeleteTaskChecklistItemInput struct {
	Account       auth.UserAccount
	TenantContext workspace.TenantContext
	ProjectID     uuid.UUID
	TaskID        uuid.UUID
	ItemID        uuid.UUID
}

type CreateTaskAttachmentUploadInput struct {
	Account       auth.UserAccount
	TenantContext workspace.TenantContext
	ProjectID     uuid.UUID
	TaskID        uuid.UUID
	FileName      string
	ContentType   string
	SizeBytes     int64
}

type CreateTaskAttachmentUploadResult struct {
	Attachment task.Attachment
	UploadURL  task.AttachmentUploadURL
}

type CompleteTaskAttachmentUploadInput struct {
	Account       auth.UserAccount
	TenantContext workspace.TenantContext
	ProjectID     uuid.UUID
	TaskID        uuid.UUID
	AttachmentID  uuid.UUID
}

type GetTaskAttachmentDownloadURLInput struct {
	Account       auth.UserAccount
	TenantContext workspace.TenantContext
	ProjectID     uuid.UUID
	TaskID        uuid.UUID
	AttachmentID  uuid.UUID
}

type GetTaskAttachmentDownloadURLResult struct {
	Attachment  task.Attachment
	DownloadURL task.AttachmentDownloadURL
}

type DeleteTaskAttachmentInput struct {
	Account       auth.UserAccount
	TenantContext workspace.TenantContext
	ProjectID     uuid.UUID
	TaskID        uuid.UUID
	AttachmentID  uuid.UUID
}

type ListTaskAttachmentsInput struct {
	Account       auth.UserAccount
	TenantContext workspace.TenantContext
	ProjectID     uuid.UUID
	TaskID        uuid.UUID
	Limit         int
	Offset        int
}

type ListTaskAttachmentsResult struct {
	Items []task.Attachment
	Total int
}

type UpdateTaskInput struct {
	Account          auth.UserAccount
	TenantContext    workspace.TenantContext
	ProjectID        uuid.UUID
	TaskID           uuid.UUID
	Title            *string
	Priority         *task.Priority
	AssigneeMemberID **uuid.UUID
	Description      *string
	StartDate        **time.Time
	DueDate          **time.Time
}

type UpdateTaskResult struct {
	Task task.Task
}

type UpdateTaskStatusInput struct {
	Account       auth.UserAccount
	TenantContext workspace.TenantContext
	ProjectID     uuid.UUID
	TaskID        uuid.UUID
	Status        task.Status
}

type UpdateTaskStatusResult struct {
	Task task.Task
}

type DeleteTaskInput struct {
	Account       auth.UserAccount
	TenantContext workspace.TenantContext
	ProjectID     uuid.UUID
	TaskID        uuid.UUID
}

type Repository interface {
	task.Repository
}

type AttachmentStorage interface {
	PresignUpload(ctx context.Context, objectKey string, contentType string) (*task.AttachmentUploadURL, error)
	PresignDownload(ctx context.Context, objectKey string, fileName string, contentType string) (*task.AttachmentDownloadURL, error)
	Bucket() string
}

type Service struct {
	repository        Repository
	attachmentStorage AttachmentStorage
	clock             func() time.Time
}

type Option func(*Service)

func WithAttachmentStorage(storage AttachmentStorage) Option {
	return func(s *Service) {
		s.attachmentStorage = storage
	}
}

func NewService(repository Repository, opts ...Option) *Service {
	service := &Service{
		repository: repository,
		clock:      func() time.Time { return time.Now().UTC() },
	}
	for _, opt := range opts {
		opt(service)
	}
	return service
}

func (s *Service) CreateTask(ctx context.Context, input CreateTaskInput) (*CreateTaskResult, error) {
	if err := validateAccount(input.Account); err != nil {
		return nil, err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return nil, err
	}
	if input.ProjectID == uuid.Nil {
		return nil, ErrProjectIDRequired
	}

	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, ErrTaskTitleRequired
	}
	description := strings.TrimSpace(input.Description)
	priorityCode := input.Priority
	if priorityCode == "" {
		priorityCode = task.PriorityMedium
	}

	now := s.clock()
	var created task.Task
	err := s.repository.WithinTransaction(ctx, func(ctx context.Context, repo task.Repository) error {
		exists, err := repo.ProjectExists(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrProjectNotFound
		}

		priority, err := repo.FindPriorityByCode(ctx, priorityCode)
		if errors.Is(err, task.ErrPriorityNotFound) {
			return ErrTaskPriorityInvalid
		}
		if err != nil {
			return err
		}

		if input.AssigneeMemberID != nil {
			if *input.AssigneeMemberID == uuid.Nil {
				return ErrTaskAssigneeNotFound
			}
			exists, err := repo.FindActiveProjectMemberByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, *input.AssigneeMemberID)
			if err != nil {
				return err
			}
			if !exists {
				return ErrTaskAssigneeNotFound
			}
		}

		taskNo, err := repo.NextTaskNo(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, defaultTaskNoPrefix, defaultTaskNoLength)
		if err != nil {
			return err
		}

		created = task.Task{
			TenantID:         input.TenantContext.TenantID,
			WorkspaceID:      input.TenantContext.WorkspaceID,
			ProjectID:        input.ProjectID,
			No:               taskNo,
			Title:            title,
			Status:           task.StatusTodo,
			PriorityID:       priority.ID,
			Priority:         priority.Code,
			AssigneeMemberID: input.AssigneeMemberID,
			Description:      description,
			StartDate:        input.StartDate,
			DueDate:          input.DueDate,
			CreatedBy:        input.Account.ID,
			CreatedAt:        now,
			UpdatedBy:        &input.Account.ID,
			UpdatedAt:        now,
		}
		if err := repo.CreateTask(ctx, &created); err != nil {
			if errors.Is(err, task.ErrTaskNoAlreadyTaken) {
				return errors.Join(ErrTaskCreateFail, err)
			}
			return err
		}
		if err := createActivity(ctx, repo, created, input.Account.ID, task.ActivityCreated, nil, &created.Status, now, map[string]any{
			"task_no": created.No,
			"title":   created.Title,
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &CreateTaskResult{Task: created}, nil
}

func (s *Service) ListTasks(ctx context.Context, input ListTasksInput) (*ListTasksResult, error) {
	if err := validateAccount(input.Account); err != nil {
		return nil, err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return nil, err
	}
	if input.ProjectID == uuid.Nil {
		return nil, ErrProjectIDRequired
	}
	limit := input.Limit
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}

	exists, err := s.repository.ProjectExists(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrProjectNotFound
	}

	filter := task.ListFilter{
		Status:           input.Status,
		AssigneeMemberID: input.AssigneeMemberID,
	}
	if input.Status != nil && !input.Status.IsValid() {
		return nil, ErrTaskStatusInvalid
	}
	if input.Priority != nil {
		if !input.Priority.IsValid() {
			return nil, ErrTaskPriorityInvalid
		}
		priority, err := s.repository.FindPriorityByCode(ctx, *input.Priority)
		if errors.Is(err, task.ErrPriorityNotFound) {
			return nil, ErrTaskPriorityInvalid
		}
		if err != nil {
			return nil, err
		}
		filter.PriorityID = &priority.ID
	}

	items, total, err := s.repository.ListTasks(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, filter, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListTasksResult{Items: items, Total: total}, nil
}

func (s *Service) GetTaskBoardSummary(ctx context.Context, input GetTaskBoardSummaryInput) (*GetTaskBoardSummaryResult, error) {
	if err := validateAccount(input.Account); err != nil {
		return nil, err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return nil, err
	}
	if input.ProjectID == uuid.Nil {
		return nil, ErrProjectIDRequired
	}

	exists, err := s.repository.ProjectExists(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrProjectNotFound
	}

	rows, err := s.repository.CountTasksByStatus(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID)
	if err != nil {
		return nil, err
	}

	countsByStatus := make(map[task.Status]int, len(task.AllStatuses()))
	for _, status := range task.AllStatuses() {
		countsByStatus[status] = 0
	}
	for _, row := range rows {
		if row.Status.IsValid() {
			countsByStatus[row.Status] = row.Count
		}
	}

	counts := make([]task.StatusCount, 0, len(task.AllStatuses()))
	total := 0
	for _, status := range task.AllStatuses() {
		count := countsByStatus[status]
		total += count
		counts = append(counts, task.StatusCount{Status: status, Count: count})
	}

	return &GetTaskBoardSummaryResult{ProjectID: input.ProjectID, Counts: counts, Total: total}, nil
}

func (s *Service) GetTask(ctx context.Context, input GetTaskInput) (*GetTaskResult, error) {
	if err := validateAccount(input.Account); err != nil {
		return nil, err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return nil, err
	}
	if input.ProjectID == uuid.Nil {
		return nil, ErrProjectIDRequired
	}
	if input.TaskID == uuid.Nil {
		return nil, ErrTaskIDRequired
	}

	item, err := s.repository.FindTaskByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID)
	if errors.Is(err, task.ErrTaskNotFound) {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	return &GetTaskResult{Task: *item}, nil
}

func (s *Service) ListTaskActivities(ctx context.Context, input ListTaskActivitiesInput) (*ListTaskActivitiesResult, error) {
	if err := validateAccount(input.Account); err != nil {
		return nil, err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return nil, err
	}
	if input.ProjectID == uuid.Nil {
		return nil, ErrProjectIDRequired
	}
	if input.TaskID == uuid.Nil {
		return nil, ErrTaskIDRequired
	}
	limit := input.Limit
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}

	if _, err := s.repository.FindTaskByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID); err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	items, total, err := s.repository.ListTaskActivities(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListTaskActivitiesResult{Items: items, Total: total}, nil
}

func (s *Service) CreateTaskComment(ctx context.Context, input CreateTaskCommentInput) (*CreateTaskCommentResult, error) {
	if err := validateAccount(input.Account); err != nil {
		return nil, err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return nil, err
	}
	if input.ProjectID == uuid.Nil {
		return nil, ErrProjectIDRequired
	}
	if input.TaskID == uuid.Nil {
		return nil, ErrTaskIDRequired
	}
	body := strings.TrimSpace(input.Body)
	if body == "" {
		return nil, ErrTaskCommentBodyRequired
	}

	now := s.clock()
	comment := task.Comment{
		TenantID:    input.TenantContext.TenantID,
		WorkspaceID: input.TenantContext.WorkspaceID,
		ProjectID:   input.ProjectID,
		TaskID:      input.TaskID,
		Body:        body,
		CreatedBy:   input.Account.ID,
		CreatedAt:   now,
		UpdatedBy:   &input.Account.ID,
		UpdatedAt:   now,
	}

	err := s.repository.WithinTransaction(ctx, func(ctx context.Context, repo task.Repository) error {
		item, err := repo.FindTaskByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID)
		if errors.Is(err, task.ErrTaskNotFound) {
			return ErrTaskNotFound
		}
		if err != nil {
			return err
		}
		if err := repo.CreateTaskComment(ctx, &comment); err != nil {
			return err
		}
		return createActivity(ctx, repo, *item, input.Account.ID, task.ActivityUpdated, nil, nil, now, map[string]any{
			"comment_id": comment.ID.String(),
			"action":     "comment_created",
		})
	})
	if err != nil {
		return nil, err
	}
	return &CreateTaskCommentResult{Comment: comment}, nil
}

func (s *Service) ListTaskComments(ctx context.Context, input ListTaskCommentsInput) (*ListTaskCommentsResult, error) {
	if err := validateAccount(input.Account); err != nil {
		return nil, err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return nil, err
	}
	if input.ProjectID == uuid.Nil {
		return nil, ErrProjectIDRequired
	}
	if input.TaskID == uuid.Nil {
		return nil, ErrTaskIDRequired
	}
	limit := input.Limit
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}

	if _, err := s.repository.FindTaskByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID); err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	items, total, err := s.repository.ListTaskComments(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListTaskCommentsResult{Items: items, Total: total}, nil
}

func (s *Service) DeleteTaskComment(ctx context.Context, input DeleteTaskCommentInput) error {
	if err := validateAccount(input.Account); err != nil {
		return err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return err
	}
	if input.ProjectID == uuid.Nil {
		return ErrProjectIDRequired
	}
	if input.TaskID == uuid.Nil {
		return ErrTaskIDRequired
	}
	if input.CommentID == uuid.Nil {
		return ErrTaskCommentNotFound
	}

	now := s.clock()
	err := s.repository.WithinTransaction(ctx, func(ctx context.Context, repo task.Repository) error {
		item, err := repo.FindTaskByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID)
		if errors.Is(err, task.ErrTaskNotFound) {
			return ErrTaskNotFound
		}
		if err != nil {
			return err
		}
		if err := repo.SoftDeleteTaskComment(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID, input.CommentID, input.Account.ID, now); err != nil {
			if errors.Is(err, task.ErrCommentNotFound) {
				return ErrTaskCommentNotFound
			}
			return err
		}
		return createActivity(ctx, repo, *item, input.Account.ID, task.ActivityUpdated, nil, nil, now, map[string]any{
			"comment_id": input.CommentID.String(),
			"action":     "comment_deleted",
		})
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) CreateTaskChecklistItem(ctx context.Context, input CreateTaskChecklistItemInput) (*CreateTaskChecklistItemResult, error) {
	if err := validateAccount(input.Account); err != nil {
		return nil, err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return nil, err
	}
	if input.ProjectID == uuid.Nil {
		return nil, ErrProjectIDRequired
	}
	if input.TaskID == uuid.Nil {
		return nil, ErrTaskIDRequired
	}
	text := strings.TrimSpace(input.Text)
	if text == "" {
		return nil, ErrTaskChecklistTextRequired
	}
	if input.SortOrder < 0 {
		return nil, ErrTaskChecklistSortOrderInvalid
	}

	now := s.clock()
	item := task.ChecklistItem{
		TenantID:    input.TenantContext.TenantID,
		WorkspaceID: input.TenantContext.WorkspaceID,
		ProjectID:   input.ProjectID,
		TaskID:      input.TaskID,
		Text:        text,
		SortOrder:   input.SortOrder,
		CreatedBy:   input.Account.ID,
		CreatedAt:   now,
		UpdatedBy:   &input.Account.ID,
		UpdatedAt:   now,
	}

	err := s.repository.WithinTransaction(ctx, func(ctx context.Context, repo task.Repository) error {
		parent, err := repo.FindTaskByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID)
		if errors.Is(err, task.ErrTaskNotFound) {
			return ErrTaskNotFound
		}
		if err != nil {
			return err
		}
		if err := repo.CreateTaskChecklistItem(ctx, &item); err != nil {
			return err
		}
		return createActivity(ctx, repo, *parent, input.Account.ID, task.ActivityUpdated, nil, nil, now, map[string]any{
			"checklist_item_id": item.ID.String(),
			"action":            "checklist_item_created",
		})
	})
	if err != nil {
		return nil, err
	}
	return &CreateTaskChecklistItemResult{Item: item}, nil
}

func (s *Service) ListTaskChecklistItems(ctx context.Context, input ListTaskChecklistItemsInput) (*ListTaskChecklistItemsResult, error) {
	if err := validateAccount(input.Account); err != nil {
		return nil, err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return nil, err
	}
	if input.ProjectID == uuid.Nil {
		return nil, ErrProjectIDRequired
	}
	if input.TaskID == uuid.Nil {
		return nil, ErrTaskIDRequired
	}
	if _, err := s.repository.FindTaskByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID); err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	items, err := s.repository.ListTaskChecklistItems(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID)
	if err != nil {
		return nil, err
	}
	return &ListTaskChecklistItemsResult{Items: items}, nil
}

func (s *Service) UpdateTaskChecklistItem(ctx context.Context, input UpdateTaskChecklistItemInput) (*UpdateTaskChecklistItemResult, error) {
	if err := validateAccount(input.Account); err != nil {
		return nil, err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return nil, err
	}
	if input.ProjectID == uuid.Nil {
		return nil, ErrProjectIDRequired
	}
	if input.TaskID == uuid.Nil {
		return nil, ErrTaskIDRequired
	}
	if input.ItemID == uuid.Nil {
		return nil, ErrTaskChecklistItemNotFound
	}

	patch := task.ChecklistItemPatch{
		UpdatedBy: input.Account.ID,
		UpdatedAt: s.clock(),
	}
	hasField := false
	if input.Text != nil {
		text := strings.TrimSpace(*input.Text)
		if text == "" {
			return nil, ErrTaskChecklistTextRequired
		}
		patch.Text = &text
		hasField = true
	}
	if input.IsCompleted != nil {
		patch.IsCompleted = input.IsCompleted
		hasField = true
	}
	if input.SortOrder != nil {
		if *input.SortOrder < 0 {
			return nil, ErrTaskChecklistSortOrderInvalid
		}
		patch.SortOrder = input.SortOrder
		hasField = true
	}
	if !hasField {
		return nil, ErrTaskChecklistUpdateNoFields
	}

	now := patch.UpdatedAt
	err := s.repository.WithinTransaction(ctx, func(ctx context.Context, repo task.Repository) error {
		parent, err := repo.FindTaskByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID)
		if errors.Is(err, task.ErrTaskNotFound) {
			return ErrTaskNotFound
		}
		if err != nil {
			return err
		}
		before, err := repo.FindTaskChecklistItemByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID, input.ItemID)
		if errors.Is(err, task.ErrChecklistItemNotFound) {
			return ErrTaskChecklistItemNotFound
		}
		if err != nil {
			return err
		}
		if err := repo.UpdateTaskChecklistItem(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID, input.ItemID, patch); err != nil {
			if errors.Is(err, task.ErrChecklistItemNotFound) {
				return ErrTaskChecklistItemNotFound
			}
			return err
		}
		return createActivity(ctx, repo, *parent, input.Account.ID, task.ActivityUpdated, nil, nil, now, map[string]any{
			"checklist_item_id": input.ItemID.String(),
			"action":            "checklist_item_updated",
			"was_completed":     before.IsCompleted,
			"is_completed":      updatedChecklistCompleted(before.IsCompleted, input.IsCompleted),
		})
	})
	if err != nil {
		return nil, err
	}
	item, err := s.repository.FindTaskChecklistItemByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID, input.ItemID)
	if errors.Is(err, task.ErrChecklistItemNotFound) {
		return nil, ErrTaskChecklistItemNotFound
	}
	if err != nil {
		return nil, err
	}
	return &UpdateTaskChecklistItemResult{Item: *item}, nil
}

func (s *Service) DeleteTaskChecklistItem(ctx context.Context, input DeleteTaskChecklistItemInput) error {
	if err := validateAccount(input.Account); err != nil {
		return err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return err
	}
	if input.ProjectID == uuid.Nil {
		return ErrProjectIDRequired
	}
	if input.TaskID == uuid.Nil {
		return ErrTaskIDRequired
	}
	if input.ItemID == uuid.Nil {
		return ErrTaskChecklistItemNotFound
	}

	now := s.clock()
	err := s.repository.WithinTransaction(ctx, func(ctx context.Context, repo task.Repository) error {
		parent, err := repo.FindTaskByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID)
		if errors.Is(err, task.ErrTaskNotFound) {
			return ErrTaskNotFound
		}
		if err != nil {
			return err
		}
		if err := repo.SoftDeleteTaskChecklistItem(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID, input.ItemID, input.Account.ID, now); err != nil {
			if errors.Is(err, task.ErrChecklistItemNotFound) {
				return ErrTaskChecklistItemNotFound
			}
			return err
		}
		return createActivity(ctx, repo, *parent, input.Account.ID, task.ActivityUpdated, nil, nil, now, map[string]any{
			"checklist_item_id": input.ItemID.String(),
			"action":            "checklist_item_deleted",
		})
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) CreateTaskAttachmentUpload(ctx context.Context, input CreateTaskAttachmentUploadInput) (*CreateTaskAttachmentUploadResult, error) {
	if err := validateAccount(input.Account); err != nil {
		return nil, err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return nil, err
	}
	if s.attachmentStorage == nil {
		return nil, ErrAttachmentStorageNotConfigured
	}
	if input.ProjectID == uuid.Nil {
		return nil, ErrProjectIDRequired
	}
	if input.TaskID == uuid.Nil {
		return nil, ErrTaskIDRequired
	}

	fileName := strings.TrimSpace(input.FileName)
	if fileName == "" {
		return nil, ErrAttachmentFileNameRequired
	}
	contentType := strings.TrimSpace(input.ContentType)
	if contentType == "" {
		return nil, ErrAttachmentContentTypeRequired
	}
	if input.SizeBytes <= 0 {
		return nil, ErrAttachmentSizeInvalid
	}

	item, err := s.repository.FindTaskByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID)
	if errors.Is(err, task.ErrTaskNotFound) {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}

	attachmentID, err := ids.NewUUID()
	if err != nil {
		return nil, err
	}
	now := s.clock()
	objectKey := buildAttachmentObjectKey(input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID, attachmentID, fileName)
	attachment := task.Attachment{
		ID:            attachmentID,
		TenantID:      input.TenantContext.TenantID,
		WorkspaceID:   input.TenantContext.WorkspaceID,
		ProjectID:     input.ProjectID,
		TaskID:        input.TaskID,
		FileName:      fileName,
		ContentType:   contentType,
		SizeBytes:     input.SizeBytes,
		StorageBucket: s.attachmentStorage.Bucket(),
		ObjectKey:     objectKey,
		UploadStatus:  task.AttachmentPending,
		UploadedBy:    input.Account.ID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	uploadURL, err := s.attachmentStorage.PresignUpload(ctx, objectKey, contentType)
	if err != nil {
		return nil, err
	}

	err = s.repository.WithinTransaction(ctx, func(ctx context.Context, repo task.Repository) error {
		if err := repo.CreateTaskAttachment(ctx, &attachment); err != nil {
			return err
		}
		return createActivity(ctx, repo, *item, input.Account.ID, task.ActivityUpdated, nil, nil, now, map[string]any{
			"attachment_id": attachment.ID.String(),
			"file_name":     attachment.FileName,
			"action":        "attachment_created",
		})
	})
	if err != nil {
		return nil, err
	}

	return &CreateTaskAttachmentUploadResult{Attachment: attachment, UploadURL: *uploadURL}, nil
}

func (s *Service) CompleteTaskAttachmentUpload(ctx context.Context, input CompleteTaskAttachmentUploadInput) error {
	if err := validateAccount(input.Account); err != nil {
		return err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return err
	}
	if input.ProjectID == uuid.Nil {
		return ErrProjectIDRequired
	}
	if input.TaskID == uuid.Nil {
		return ErrTaskIDRequired
	}
	if input.AttachmentID == uuid.Nil {
		return ErrTaskAttachmentNotFound
	}
	now := s.clock()

	err := s.repository.WithinTransaction(ctx, func(ctx context.Context, repo task.Repository) error {
		item, err := repo.FindTaskByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID)
		if errors.Is(err, task.ErrTaskNotFound) {
			return ErrTaskNotFound
		}
		if err != nil {
			return err
		}

		if err := repo.MarkTaskAttachmentUploaded(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID, input.AttachmentID, now); err != nil {
			if errors.Is(err, task.ErrAttachmentNotFound) {
				return ErrTaskAttachmentNotFound
			}
			return err
		}

		return createActivity(ctx, repo, *item, input.Account.ID, task.ActivityUpdated, nil, nil, now, map[string]any{
			"attachment_id": input.AttachmentID.String(),
			"action":        "attachment_uploaded",
		})
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) GetTaskAttachmentDownloadURL(ctx context.Context, input GetTaskAttachmentDownloadURLInput) (*GetTaskAttachmentDownloadURLResult, error) {
	if err := validateAccount(input.Account); err != nil {
		return nil, err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return nil, err
	}
	if s.attachmentStorage == nil {
		return nil, ErrAttachmentStorageNotConfigured
	}
	if input.ProjectID == uuid.Nil {
		return nil, ErrProjectIDRequired
	}
	if input.TaskID == uuid.Nil {
		return nil, ErrTaskIDRequired
	}
	if input.AttachmentID == uuid.Nil {
		return nil, ErrTaskAttachmentNotFound
	}
	if _, err := s.repository.FindTaskByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID); err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	attachment, err := s.repository.FindTaskAttachmentByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID, input.AttachmentID)
	if errors.Is(err, task.ErrAttachmentNotFound) {
		return nil, ErrTaskAttachmentNotFound
	}
	if err != nil {
		return nil, err
	}
	if attachment.UploadStatus != task.AttachmentUploaded {
		return nil, ErrAttachmentNotUploaded
	}

	downloadURL, err := s.attachmentStorage.PresignDownload(ctx, attachment.ObjectKey, attachment.FileName, attachment.ContentType)
	if err != nil {
		return nil, err
	}
	return &GetTaskAttachmentDownloadURLResult{Attachment: *attachment, DownloadURL: *downloadURL}, nil
}

func (s *Service) ListTaskAttachments(ctx context.Context, input ListTaskAttachmentsInput) (*ListTaskAttachmentsResult, error) {
	if err := validateAccount(input.Account); err != nil {
		return nil, err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return nil, err
	}
	if input.ProjectID == uuid.Nil {
		return nil, ErrProjectIDRequired
	}
	if input.TaskID == uuid.Nil {
		return nil, ErrTaskIDRequired
	}
	limit := input.Limit
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}
	if _, err := s.repository.FindTaskByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID); err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	items, total, err := s.repository.ListTaskAttachments(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListTaskAttachmentsResult{Items: items, Total: total}, nil
}

func (s *Service) DeleteTaskAttachment(ctx context.Context, input DeleteTaskAttachmentInput) error {
	if err := validateAccount(input.Account); err != nil {
		return err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return err
	}
	if input.ProjectID == uuid.Nil {
		return ErrProjectIDRequired
	}
	if input.TaskID == uuid.Nil {
		return ErrTaskIDRequired
	}
	if input.AttachmentID == uuid.Nil {
		return ErrTaskAttachmentNotFound
	}

	now := s.clock()
	err := s.repository.WithinTransaction(ctx, func(ctx context.Context, repo task.Repository) error {
		item, err := repo.FindTaskByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID)
		if errors.Is(err, task.ErrTaskNotFound) {
			return ErrTaskNotFound
		}
		if err != nil {
			return err
		}
		if err := repo.SoftDeleteTaskAttachment(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID, input.AttachmentID, input.Account.ID, now); err != nil {
			if errors.Is(err, task.ErrAttachmentNotFound) {
				return ErrTaskAttachmentNotFound
			}
			return err
		}
		return createActivity(ctx, repo, *item, input.Account.ID, task.ActivityUpdated, nil, nil, now, map[string]any{
			"attachment_id": input.AttachmentID.String(),
			"action":        "attachment_deleted",
		})
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) UpdateTask(ctx context.Context, input UpdateTaskInput) (*UpdateTaskResult, error) {
	if err := validateAccount(input.Account); err != nil {
		return nil, err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return nil, err
	}
	if input.ProjectID == uuid.Nil {
		return nil, ErrProjectIDRequired
	}
	if input.TaskID == uuid.Nil {
		return nil, ErrTaskIDRequired
	}

	patch := task.Patch{
		UpdatedBy: input.Account.ID,
		UpdatedAt: s.clock(),
	}
	hasField := false

	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if title == "" {
			return nil, ErrTaskTitleRequired
		}
		patch.Title = &title
		hasField = true
	}
	if input.Priority != nil {
		priority, err := s.repository.FindPriorityByCode(ctx, *input.Priority)
		if errors.Is(err, task.ErrPriorityNotFound) {
			return nil, ErrTaskPriorityInvalid
		}
		if err != nil {
			return nil, err
		}
		patch.PriorityID = &priority.ID
		hasField = true
	}
	if input.AssigneeMemberID != nil {
		if *input.AssigneeMemberID != nil {
			if **input.AssigneeMemberID == uuid.Nil {
				return nil, ErrTaskAssigneeNotFound
			}
			exists, err := s.repository.FindActiveProjectMemberByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, **input.AssigneeMemberID)
			if err != nil {
				return nil, err
			}
			if !exists {
				return nil, ErrTaskAssigneeNotFound
			}
		}
		patch.AssigneeMemberID = input.AssigneeMemberID
		hasField = true
	}
	if input.Description != nil {
		description := strings.TrimSpace(*input.Description)
		patch.Description = &description
		hasField = true
	}
	if input.StartDate != nil {
		patch.StartDate = input.StartDate
		hasField = true
	}
	if input.DueDate != nil {
		patch.DueDate = input.DueDate
		hasField = true
	}
	if !hasField {
		return nil, ErrTaskUpdateNoFields
	}

	now := patch.UpdatedAt
	err := s.repository.WithinTransaction(ctx, func(ctx context.Context, repo task.Repository) error {
		before, err := repo.FindTaskByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID)
		if errors.Is(err, task.ErrTaskNotFound) {
			return ErrTaskNotFound
		}
		if err != nil {
			return err
		}

		if err := repo.UpdateTask(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID, patch); err != nil {
			if errors.Is(err, task.ErrTaskNotFound) {
				return ErrTaskNotFound
			}
			return err
		}

		return createActivity(ctx, repo, *before, input.Account.ID, task.ActivityUpdated, nil, nil, now, map[string]any{
			"changed_fields": changedFields(input),
		})
	})
	if err != nil {
		return nil, err
	}
	item, err := s.repository.FindTaskByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID)
	if errors.Is(err, task.ErrTaskNotFound) {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	return &UpdateTaskResult{Task: *item}, nil
}

func (s *Service) UpdateTaskStatus(ctx context.Context, input UpdateTaskStatusInput) (*UpdateTaskStatusResult, error) {
	if err := validateAccount(input.Account); err != nil {
		return nil, err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return nil, err
	}
	if input.ProjectID == uuid.Nil {
		return nil, ErrProjectIDRequired
	}
	if input.TaskID == uuid.Nil {
		return nil, ErrTaskIDRequired
	}
	if !input.Status.IsValid() {
		return nil, ErrTaskStatusInvalid
	}

	now := s.clock()
	var completedDate *time.Time
	if input.Status == task.StatusDone {
		completedDate = &now
	}

	err := s.repository.WithinTransaction(ctx, func(ctx context.Context, repo task.Repository) error {
		before, err := repo.FindTaskByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID)
		if errors.Is(err, task.ErrTaskNotFound) {
			return ErrTaskNotFound
		}
		if err != nil {
			return err
		}

		if err := repo.UpdateTaskStatus(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID, input.Status, completedDate, input.Account.ID, now); err != nil {
			if errors.Is(err, task.ErrTaskNotFound) {
				return ErrTaskNotFound
			}
			return err
		}
		return createActivity(ctx, repo, *before, input.Account.ID, task.ActivityStatusChanged, &before.Status, &input.Status, now, nil)
	})
	if err != nil {
		return nil, err
	}

	item, err := s.repository.FindTaskByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID)
	if errors.Is(err, task.ErrTaskNotFound) {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	return &UpdateTaskStatusResult{Task: *item}, nil
}

func (s *Service) DeleteTask(ctx context.Context, input DeleteTaskInput) error {
	if err := validateAccount(input.Account); err != nil {
		return err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return err
	}
	if input.ProjectID == uuid.Nil {
		return ErrProjectIDRequired
	}
	if input.TaskID == uuid.Nil {
		return ErrTaskIDRequired
	}

	now := s.clock()
	err := s.repository.WithinTransaction(ctx, func(ctx context.Context, repo task.Repository) error {
		before, err := repo.FindTaskByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID)
		if errors.Is(err, task.ErrTaskNotFound) {
			return ErrTaskNotFound
		}
		if err != nil {
			return err
		}
		if err := repo.SoftDeleteTask(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID, input.Account.ID); err != nil {
			if errors.Is(err, task.ErrTaskNotFound) {
				return ErrTaskNotFound
			}
			return err
		}
		return createActivity(ctx, repo, *before, input.Account.ID, task.ActivityDeleted, &before.Status, nil, now, nil)
	})
	if err != nil {
		return err
	}
	return nil
}

func createActivity(ctx context.Context, repo task.Repository, item task.Task, actorID uuid.UUID, action task.ActivityAction, fromStatus *task.Status, toStatus *task.Status, createdAt time.Time, metadata map[string]any) error {
	return repo.CreateTaskActivity(ctx, &task.Activity{
		TenantID:       item.TenantID,
		WorkspaceID:    item.WorkspaceID,
		ProjectID:      item.ProjectID,
		TaskID:         item.ID,
		ActorAccountID: actorID,
		Action:         action,
		FromStatus:     fromStatus,
		ToStatus:       toStatus,
		MetadataJSON:   metadata,
		CreatedAt:      createdAt,
	})
}

func updatedChecklistCompleted(current bool, next *bool) bool {
	if next == nil {
		return current
	}
	return *next
}

func changedFields(input UpdateTaskInput) []string {
	fields := make([]string, 0, 6)
	if input.Title != nil {
		fields = append(fields, "title")
	}
	if input.Priority != nil {
		fields = append(fields, "priority")
	}
	if input.AssigneeMemberID != nil {
		fields = append(fields, "assignee_member_id")
	}
	if input.Description != nil {
		fields = append(fields, "description")
	}
	if input.StartDate != nil {
		fields = append(fields, "start_date")
	}
	if input.DueDate != nil {
		fields = append(fields, "due_date")
	}
	return fields
}

func buildAttachmentObjectKey(tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, attachmentID uuid.UUID, fileName string) string {
	return path.Join(
		"tenants", tenantID.String(),
		"workspaces", workspaceID.String(),
		"projects", projectID.String(),
		"tasks", taskID.String(),
		"attachments", attachmentID.String(),
		sanitizeFileName(fileName),
	)
}

func sanitizeFileName(value string) string {
	value = path.Base(strings.TrimSpace(value))
	if value == "." || value == "/" || value == "" {
		return "attachment"
	}
	var builder strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '.', r == '-', r == '_':
			builder.WriteRune(r)
		default:
			builder.WriteRune('_')
		}
	}
	sanitized := strings.Trim(builder.String(), "._-")
	if sanitized == "" {
		return "attachment"
	}
	return sanitized
}

func validateAccount(account auth.UserAccount) error {
	if account.ID == uuid.Nil {
		return ErrAccountRequired
	}
	if account.Status != auth.UserAccountStatusActive {
		return ErrAccountInactive
	}
	return nil
}

func validateTenantContext(context workspace.TenantContext) error {
	if context.TenantID == uuid.Nil || context.WorkspaceID == uuid.Nil || context.MembershipID == uuid.Nil {
		return ErrTenantContextRequired
	}
	return nil
}
