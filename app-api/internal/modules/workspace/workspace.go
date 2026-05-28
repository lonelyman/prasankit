package workspace

import (
	"time"

	"github.com/google/uuid"
)

// Workspace is the pure domain representation of a workspaces row.
// No gorm tags — mapping happens in adapters/database/workspace.
type Workspace struct {
	ID                  uuid.UUID
	WorkspaceName       string
	Slug                string
	WorkspaceStatusCode string
	ContactEmail        string
	OwnerUserAccountID  uuid.UUID
	CreatedAt           time.Time
	CreatedBy           *uuid.UUID
	UpdatedAt           time.Time
	UpdatedBy           *uuid.UUID
	PendingDeletionAt   *time.Time
	DeletedAt           *time.Time
	DeletedBy           *uuid.UUID
}

// WorkspaceWithRole bundles a workspace with the requesting account's role,
// used in list responses.
type WorkspaceWithRole struct {
	Workspace   Workspace
	OrgRoleCode string
}
