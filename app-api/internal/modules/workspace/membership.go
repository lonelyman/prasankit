package workspace

import (
	"time"

	"github.com/google/uuid"
)

// Membership is the pure domain representation of a workspace_memberships row.
type Membership struct {
	ID                     uuid.UUID
	WorkspaceID            uuid.UUID
	UserAccountID          uuid.UUID
	OrgRoleCode            string
	MembershipStatusCode   string
	InvitedByUserAccountID *uuid.UUID
	JoinedAt               *time.Time
	CreatedAt              time.Time
	CreatedBy              *uuid.UUID
	UpdatedAt              time.Time
	UpdatedBy              *uuid.UUID
}

// TenantContext carries the resolved workspace + membership context for a request.
// It is constructed by the requireTenantContext middleware and injected into c.Locals.
// Service methods receive workspace_id exclusively from here (never from client input).
type TenantContext struct {
	WorkspaceID  uuid.UUID
	MembershipID uuid.UUID
	AccountID    uuid.UUID
	OrgRoleCode  string
}
