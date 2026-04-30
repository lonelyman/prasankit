package tasksvc

import (
	"context"
	"errors"
	"strings"
	"time"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/task"
	"prasankit-api/internal/modules/workspace"

	"github.com/google/uuid"
)

const (
	defaultTaskNoPrefix = "TASK"
	defaultTaskNoLength = 4
)

var (
	ErrAccountRequired       = errors.New("account is required")
	ErrAccountInactive       = errors.New("account is inactive")
	ErrTenantContextRequired = errors.New("tenant context is required")
	ErrProjectIDRequired     = errors.New("project id is required")
	ErrProjectNotFound       = errors.New("project not found")
	ErrTaskIDRequired        = errors.New("task id is required")
	ErrTaskNotFound          = errors.New("task not found")
	ErrTaskTitleRequired     = errors.New("task title is required")
	ErrTaskUpdateNoFields    = errors.New("task update has no fields")
	ErrTaskStatusInvalid     = errors.New("task status is invalid")
	ErrTaskPriorityInvalid   = errors.New("task priority is invalid")
	ErrTaskAssigneeNotFound  = errors.New("task assignee not found")
	ErrTaskCreateFail        = errors.New("task create failed")
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

type GetTaskInput struct {
	Account       auth.UserAccount
	TenantContext workspace.TenantContext
	ProjectID     uuid.UUID
	TaskID        uuid.UUID
}

type GetTaskResult struct {
	Task task.Task
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

type Service struct {
	repository Repository
	clock      func() time.Time
}

func NewService(repository Repository) *Service {
	return &Service{
		repository: repository,
		clock:      func() time.Time { return time.Now().UTC() },
	}
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

	if err := s.repository.UpdateTask(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID, patch); err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			return nil, ErrTaskNotFound
		}
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

	if err := s.repository.UpdateTaskStatus(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID, input.Status, completedDate, input.Account.ID, now); err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			return nil, ErrTaskNotFound
		}
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

	if err := s.repository.SoftDeleteTask(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.TaskID, input.Account.ID); err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			return ErrTaskNotFound
		}
		return err
	}
	return nil
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
