package projectmemberposition

import (
	"context"

	"prasankit-api/internal/modules/audit"

	"github.com/google/uuid"
)

// ProjectMemberPositionRepository: append/delete-only storage for the junction.
type ProjectMemberPositionRepository interface {
	// AssignWithAudit inserts a junction row + audit row in one tx.
	// uq_pmp_member_position violation -> ErrAlreadyAssigned; any 23503 -> ErrInvalidPositionCode
	// (the member composite FK is service-pre-guarded; a member-FK 23503 is a should-not-happen
	// race backstop — the adapter does NOT distinguish the two FKs by name).
	AssignWithAudit(ctx context.Context, mp ProjectMemberPosition, entry audit.Entry) error
	// UnassignWithAudit hard-DELETEs the junction row + audit row in one tx.
	// RowsAffected==0 -> ErrAssignmentNotFound.
	UnassignWithAudit(ctx context.Context, workspaceID, projectMemberID uuid.UUID, positionCode string, entry audit.Entry) error
	// ListByMember returns the member's positions, JOINed to project_positions (same-ws), with labels.
	ListByMember(ctx context.Context, workspaceID, projectMemberID uuid.UUID) ([]PositionWithLabel, error)
}
