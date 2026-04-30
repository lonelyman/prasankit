package task

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in_progress"
	StatusBlocked    Status = "blocked"
	StatusDone       Status = "done"
	StatusCancelled  Status = "cancelled"
)

type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

type Task struct {
	ID               uuid.UUID
	TenantID         uuid.UUID
	WorkspaceID      uuid.UUID
	ProjectID        uuid.UUID
	No               string
	Title            string
	Status           Status
	PriorityID       uuid.UUID
	Priority         Priority
	AssigneeMemberID *uuid.UUID
	Description      string
	StartDate        *time.Time
	DueDate          *time.Time
	CompletedDate    *time.Time
	CreatedBy        uuid.UUID
	CreatedAt        time.Time
	UpdatedBy        *uuid.UUID
	UpdatedAt        time.Time
	DeletedBy        *uuid.UUID
	DeletedAt        *time.Time
}

type PriorityMaster struct {
	ID   uuid.UUID
	Code Priority
	Name string
}
