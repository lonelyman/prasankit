package project

import (
	"context"

	"prasankit-api/internal/modules/audit"

	"github.com/google/uuid"
)

// ProjectRepository defines storage operations on projects.
type ProjectRepository interface {
	CreateWithAudit(ctx context.Context, p Project, entry audit.Entry) error
	// CreateWithOwner atomically, in ONE transaction (D40 3-step + D31 audit):
	//   1. INSERT projects (owner_project_member_id = NULL)        -- NULL, no FK check
	//   2. INSERT project_members (owner seed, project_role_code = project_owner)
	//   3. UPDATE projects SET owner_project_member_id = owner.ID  -- target now exists
	//   4. LogTx(projectEntry)  5. LogTx(memberEntry)
	// Returns project.ErrSlugTaken on the projects unique violation.
	CreateWithOwner(ctx context.Context, p Project, owner OwnerMemberSeed, projectEntry, memberEntry audit.Entry) error
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
	IsActiveProjectRoleCode(ctx context.Context, code string) (bool, error)
}
