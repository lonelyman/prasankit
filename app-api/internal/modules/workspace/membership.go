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

// MembershipWithDisplayName is the read projection for ListWorkspaceMembers
// (JOIN workspace_memberships → user_accounts). It carries the membership id,
// the user's display_name, and the org_role_code — exactly what the FE add-member
// picker needs (no email / PII beyond the display name).
type MembershipWithDisplayName struct {
	ID          uuid.UUID
	DisplayName string
	OrgRoleCode string
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
