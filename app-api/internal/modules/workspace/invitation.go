package workspace

import (
	"time"

	"github.com/google/uuid"
)

// Invitation is the pure domain representation of a workspace_invitations row.
// State is derived from timestamps — there is no status column.
// No gorm tags — mapping happens in adapters/database/workspace.
type Invitation struct {
	ID                     uuid.UUID
	WorkspaceID            uuid.UUID
	Email                  string
	OrgRoleCode            string
	TokenHash              string
	InvitedByUserAccountID uuid.UUID
	ExpiresAt              time.Time
	AcceptedAt             *time.Time
	RevokedAt              *time.Time
	AcceptedUserAccountID  *uuid.UUID
	CreatedAt              time.Time
}

// IsActive returns true when the invitation has not been accepted, revoked, or expired.
func (inv *Invitation) IsActive(now time.Time) bool {
	return inv.AcceptedAt == nil && inv.RevokedAt == nil && inv.ExpiresAt.After(now)
}
