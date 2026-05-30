package project

import (
	"context"

	"prasankit-api/internal/modules/audit"

	"github.com/google/uuid"
)

// ProjectRepository defines storage operations on projects.
type ProjectRepository interface {
	CreateWithAudit(ctx context.Context, p Project, entry audit.Entry) error
	FindByIDForWorkspace(ctx context.Context, workspaceID, projectID uuid.UUID) (*Project, error)
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, opts ListOptions) ([]Project, int64, error)
	UpdateWithAudit(ctx context.Context, p Project, entry audit.Entry) error
	SoftDeleteWithAudit(ctx context.Context, workspaceID, projectID, deletedBy uuid.UUID, entry audit.Entry) error
	ChangeStatusWithAudit(ctx context.Context, workspaceID, projectID uuid.UUID, newStatusCode string, updatedBy uuid.UUID, entry audit.Entry) error
}

// ListOptions carries pagination + filter parameters for ListByWorkspace.
type ListOptions struct {
	Page       int    // >= 1
	Limit      int    // [1..100]
	StatusCode string // "" → no filter
	TypeCode   string // "" → no filter
}

// MasterRepository pre-checks master vocabulary by code (D28).
type MasterRepository interface {
	IsActiveProjectStatusCode(ctx context.Context, code string) (bool, error)
	IsActiveProjectTypeCode(ctx context.Context, code string) (bool, error)
}
