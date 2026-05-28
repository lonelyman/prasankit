package workspace

import (
	"context"

	"prasankit-api/internal/modules/audit"

	"github.com/google/uuid"
)

// WorkspaceRepository defines storage operations on workspaces.
type WorkspaceRepository interface {
	// CreateWorkspaceWithOwner atomically inserts a workspace, owner membership, and audit
	// log entry in a single DB transaction. All-or-nothing: any failure rolls back all three.
	CreateWorkspaceWithOwner(ctx context.Context, ws Workspace, m Membership, entry audit.Entry) error

	// FindBySlugActive returns an active (non-deleted) workspace by slug.
	// Returns nil, nil when not found.
	FindBySlugActive(ctx context.Context, slug string) (*Workspace, error)
}

// MembershipRepository defines storage operations on workspace_memberships.
type MembershipRepository interface {
	// FindActiveByWorkspaceAndAccount returns the active membership for the given
	// workspace + account pair. Returns nil, nil when not found.
	FindActiveByWorkspaceAndAccount(ctx context.Context, workspaceID, accountID uuid.UUID) (*Membership, error)

	// ListActiveWorkspacesByAccount returns all workspaces (with role) where the account
	// has an active membership and the workspace itself is active and not deleted.
	ListActiveWorkspacesByAccount(ctx context.Context, accountID uuid.UUID) ([]WorkspaceWithRole, error)

	// ListByWorkspace returns all memberships for the given workspace_id.
	// Used for admin listing and isolation tests.
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]Membership, error)
}
