package task

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrProjectNotFound = errors.New("project not found")
var ErrTaskNotFound = errors.New("task not found")
var ErrPriorityNotFound = errors.New("task priority not found")
var ErrAssigneeNotFound = errors.New("task assignee not found")
var ErrTaskNoAlreadyTaken = errors.New("task no is already taken")

type Repository interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context, repo Repository) error) error
	ProjectExists(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID) (bool, error)
	FindPriorityByCode(ctx context.Context, code Priority) (*PriorityMaster, error)
	FindActiveProjectMemberByID(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, memberID uuid.UUID) (bool, error)
	NextTaskNo(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, prefix string, numberLength int) (string, error)
	CreateTask(ctx context.Context, item *Task) error
	UpdateTask(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, patch Patch) error
	UpdateTaskStatus(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, status Status, completedDate *time.Time, updatedBy uuid.UUID, updatedAt time.Time) error
	SoftDeleteTask(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, deletedBy uuid.UUID) error
	FindTaskByID(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID) (*Task, error)
	ListTasks(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, filter ListFilter, limit int, offset int) ([]Task, int, error)
	CountTasksByStatus(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID) ([]StatusCount, error)
}
