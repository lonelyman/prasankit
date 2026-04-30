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
	Account       auth.UserAccount
	TenantContext workspace.TenantContext
	ProjectID     uuid.UUID
	Limit         int
	Offset        int
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

	items, total, err := s.repository.ListTasks(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, limit, offset)
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
