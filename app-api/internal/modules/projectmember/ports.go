package projectmember

import (
	"context"

	"prasankit-api/internal/modules/audit"

	"github.com/google/uuid"
)

// ProjectMemberRepository defines storage operations on project_members.
type ProjectMemberRepository interface {
	// AddWithAudit inserts a project_member + audit row in one tx.
	// uq_project_members_active violation -> ErrAlreadyMember.
	// composite FK violation (cross-ws membership / project) -> propagates wrapped (DB backstop).
	AddWithAudit(ctx context.Context, m ProjectMember, entry audit.Entry) error

	// RemoveWithAudit sets removed_at=now WHERE workspace_id+project_id+id AND removed_at IS NULL,
	// + audit row, in one tx. RowsAffected==0 -> ErrProjectMemberNotFound.
	// MUST update only {removed_at, updated_at, updated_by} (never FK columns) — see Adapter note.
	RemoveWithAudit(ctx context.Context, workspaceID, projectID, memberID, removedBy uuid.UUID, entry audit.Entry) error

	// ChangeRoleWithAudit updates project_role_code WHERE workspace_id+project_id+id AND removed_at IS NULL,
	// + audit row, in one tx. FK violation -> ErrInvalidRoleCode. RowsAffected==0 -> ErrProjectMemberNotFound.
	// MUST update only {project_role_code, updated_at, updated_by} (never FK columns) — see Adapter note.
	ChangeRoleWithAudit(ctx context.Context, workspaceID, projectID, memberID uuid.UUID, newRoleCode string, updatedBy uuid.UUID, entry audit.Entry) error

	// ListByProject returns active members (removed_at IS NULL AND the membership is ws-active),
	// JOINed to workspace_memberships + user_accounts for display_name, scoped to the project's workspace.
	ListByProject(ctx context.Context, workspaceID, projectID uuid.UUID) ([]MemberWithDisplayName, error)

	// FindActiveByID returns the active project_member (removed_at IS NULL) for change-role/remove
	// pre-checks. Returns nil,nil when not found. Scoped to workspace_id.
	FindActiveByID(ctx context.Context, workspaceID, projectID, memberID uuid.UUID) (*ProjectMember, error)

	// FindByID returns the project_member INCLUDING removed rows (RemovedAt populated when
	// removed_at IS NOT NULL); returns nil,nil ONLY when the member truly does not exist in
	// this (ws, project). Read-only, additive (6b-2): the junction uses it to branch
	// nil->404 / RemovedAt!=nil->422 (the composite FK cannot catch a removed member, §M2.3.4).
	// = FindActiveByID minus the removed_at IS NULL predicate. Scoped to workspace_id.
	FindByID(ctx context.Context, workspaceID, projectID, memberID uuid.UUID) (*ProjectMember, error)

	// IsActiveWorkspaceMember verifies the target workspace_membership_id is an ACTIVE membership
	// of this workspace (membership_status_code='active'), used by AddMember before insert.
	// Composite FK is the backstop for cross-ws; this is the friendly 422 path + the active-status check
	// the FK cannot express. Returns false when not found or not active.
	IsActiveWorkspaceMember(ctx context.Context, workspaceID, membershipID uuid.UUID) (bool, error)

	// ProjectExistsForWorkspace returns true when a non-deleted project exists in this workspace.
	// Used to collapse "no such project" to 404 (D42) before touching members.
	ProjectExistsForWorkspace(ctx context.Context, workspaceID, projectID uuid.UUID) (bool, error)
}
