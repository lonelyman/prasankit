package companyposition

import (
	"context"

	"prasankit-api/internal/modules/audit"

	"github.com/google/uuid"
)

// CompanyPositionRepository: CRUD storage for the company_positions master.
type CompanyPositionRepository interface {
	// CreateWithAudit inserts a company_position + audit row in one tx.
	// uq_company_positions_workspace_code violation -> ErrCodeTaken.
	CreateWithAudit(ctx context.Context, p CompanyPosition, entry audit.Entry) error
	// FindByCodeForWorkspace returns the row keyed on (workspace_id, code), covering
	// active AND deprecated. Returns nil,nil when none.
	FindByCodeForWorkspace(ctx context.Context, workspaceID uuid.UUID, code string) (*CompanyPosition, error)
	// ListByWorkspace returns the page of positions for the workspace (+ optional status filter).
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, opts ListOptions) ([]CompanyPosition, int64, error)
	// UpdateWithAudit updates ONLY {label_th,label_en,description,sort_order,status,updated_at,updated_by}
	// keyed on (workspace_id, code) — never code/workspace_id/id/created_*.
	UpdateWithAudit(ctx context.Context, p CompanyPosition, entry audit.Entry) error
	// DeprecateWithAudit sets status='deprecated' keyed on (workspace_id, code) + audit, in one tx.
	DeprecateWithAudit(ctx context.Context, workspaceID uuid.UUID, code string, updatedBy uuid.UUID, entry audit.Entry) error
}

// ListOptions carries pagination + filter parameters for ListByWorkspace.
type ListOptions struct {
	Page   int    // >= 1
	Limit  int    // [1..100]
	Status string // "" -> no filter
}

// CompanyPositionMasterRepository is the D28 active-only pre-check for the ws-attach write.
// Domain owns the port (02 §3); companypositiondbrepo.MasterRepo implements it.
type CompanyPositionMasterRepository interface {
	IsActiveCompanyPositionCode(ctx context.Context, workspaceID uuid.UUID, code string) (bool, error)
}

// MembershipPositionRepository is the UPDATE-only writer for workspace_memberships.company_position_code
// (D41 attach). It writes via a file-private membershipRow (NO workspacedbrepo import, 02 §3),
// gating on membership_status_code='active' (mirrors projectmember.IsActiveWorkspaceMember).
// RowsAffected==0 -> ErrMembershipNotFound. 23503 -> ErrInvalidPositionCode (FK backstop).
type MembershipPositionRepository interface {
	SetCompanyPositionWithAudit(ctx context.Context, workspaceID, membershipID uuid.UUID, code *string, updatedBy uuid.UUID, entry audit.Entry) error
}
