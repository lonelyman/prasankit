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

	// ListActiveWithDisplayName returns the ACTIVE memberships of the given workspace,
	// JOINed to user_accounts for the display_name. Scoped by workspace_id (isolation).
	// Used by the add-member picker.
	ListActiveWithDisplayName(ctx context.Context, workspaceID uuid.UUID) ([]MembershipWithDisplayName, error)
}

// InvitationRepository defines storage operations on workspace_invitations.
type InvitationRepository interface {
	// CreateInvitationTx atomically inserts an invitation + audit entry in one DB transaction.
	CreateInvitationTx(ctx context.Context, inv Invitation, entry audit.Entry) error

	// FindActiveByTokenHash returns the invitation with the given token_hash that has
	// not yet been accepted, revoked, or expired. Returns nil, nil when not found.
	FindActiveByTokenHash(ctx context.Context, tokenHash string) (*Invitation, error)

	// AcceptInvitationTx atomically inserts a membership, marks the invitation as accepted,
	// and writes an audit entry — all in one DB transaction.
	AcceptInvitationTx(ctx context.Context, inv Invitation, m Membership, entry audit.Entry) error

	// IsActiveMemberByEmail returns true when there is already an active membership for
	// the given workspace + email combination (via JOIN on user_accounts.primary_email).
	// Required for the already-member pre-check at invite time (account may not exist yet,
	// so we cannot use FindActiveByWorkspaceAndAccount which needs an accountID).
	IsActiveMemberByEmail(ctx context.Context, workspaceID uuid.UUID, email string) (bool, error)
}
