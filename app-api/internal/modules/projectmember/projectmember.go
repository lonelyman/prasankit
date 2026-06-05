// Package projectmember contains the domain and service layer for the
// project_members entity. Domain structs and interfaces are pure — no gorm tags,
// no fiber imports.
package projectmember

import (
	"time"

	"github.com/google/uuid"
)

// ProjectMember is the pure domain representation of a project_members row.
// No gorm tags — mapping happens in adapters/database/projectmember.
type ProjectMember struct {
	ID                    uuid.UUID
	WorkspaceID           uuid.UUID
	ProjectID             uuid.UUID
	WorkspaceMembershipID uuid.UUID
	ProjectRoleCode       string
	JoinedAt              time.Time
	RemovedAt             *time.Time // nil = active
	CreatedAt             time.Time
	CreatedBy             uuid.UUID
	UpdatedAt             time.Time
	UpdatedBy             *uuid.UUID
}

// MemberWithDisplayName is the read projection for ListMembers (JOIN to user_accounts).
type MemberWithDisplayName struct {
	ProjectMember
	DisplayName string // from user_accounts.display_name of the membership IN THIS workspace
	// CompanyPositionCode is the membership's workspace-level company position (1:1,
	// nil = unset). Read-only projection so the FE can show it without a second call.
	CompanyPositionCode *string
}
