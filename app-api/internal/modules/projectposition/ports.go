package projectposition

import (
	"context"

	"prasankit-api/internal/modules/audit"

	"github.com/google/uuid"
)

// ProjectPositionRepository: CRUD storage for the project_positions master.
type ProjectPositionRepository interface {
	// CreateWithAudit inserts a project_position + audit row in one tx.
	// uq_project_positions_workspace_code violation -> ErrCodeTaken.
	CreateWithAudit(ctx context.Context, p ProjectPosition, entry audit.Entry) error
	// FindByCodeForWorkspace returns the row keyed on (workspace_id, code), covering
	// active AND deprecated. Returns nil,nil when none.
	FindByCodeForWorkspace(ctx context.Context, workspaceID uuid.UUID, code string) (*ProjectPosition, error)
	// ListByWorkspace returns the page of positions for the workspace (+ optional status filter).
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, opts ListOptions) ([]ProjectPosition, int64, error)
	// UpdateWithAudit updates ONLY {label_th,label_en,description,sort_order,status,updated_at,updated_by}
	// keyed on (workspace_id, code) — never code/workspace_id/id/created_*.
	UpdateWithAudit(ctx context.Context, p ProjectPosition, entry audit.Entry) error
	// DeprecateWithAudit sets status='deprecated' keyed on (workspace_id, code) + audit, in one tx.
	DeprecateWithAudit(ctx context.Context, workspaceID uuid.UUID, code string, updatedBy uuid.UUID, entry audit.Entry) error
}

// ListOptions carries pagination + filter parameters for ListByWorkspace.
type ListOptions struct {
	Page   int    // >= 1
	Limit  int    // [1..100]
	Status string // "" -> no filter
}

// PositionMasterRepository is the D28 active-only pre-check the JUNCTION consumes.
// Domain owns the port (02 §3); projectpositiondbrepo.MasterRepo implements it; the junction
// service depends on THIS interface, never the concrete adapter. Mirrors project.MasterRepository.
type PositionMasterRepository interface {
	IsActiveProjectPositionCode(ctx context.Context, workspaceID uuid.UUID, code string) (bool, error)
}
