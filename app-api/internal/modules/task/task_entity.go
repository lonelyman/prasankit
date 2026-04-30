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

func (s Status) IsValid() bool {
	switch s {
	case StatusTodo, StatusInProgress, StatusBlocked, StatusDone, StatusCancelled:
		return true
	default:
		return false
	}
}

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

type Patch struct {
	Title            *string
	PriorityID       *uuid.UUID
	AssigneeMemberID **uuid.UUID
	Description      *string
	StartDate        **time.Time
	DueDate          **time.Time
	UpdatedBy        uuid.UUID
	UpdatedAt        time.Time
}

type PriorityMaster struct {
	ID   uuid.UUID
	Code Priority
	Name string
}
