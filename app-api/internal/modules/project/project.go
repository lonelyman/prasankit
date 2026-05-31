// Package project contains the domain and service layer for the projects entity.
// Domain structs and interfaces are pure — no gorm tags, no fiber imports.
package project

import (
	"time"

	"github.com/google/uuid"
)

// Project is the pure domain representation of a projects row.
// No gorm tags — mapping happens in adapters/database/project.
type Project struct {
	ID                   uuid.UUID
	WorkspaceID          uuid.UUID
	ProjectName          string
	Slug                 *string // NULL when absent
	ProjectTypeCode      string
	ProjectStatusCode    string
	OwnerProjectMemberID *uuid.UUID // ALWAYS nil in 6a; 6b populates
	RequestingUnit       *string
	Description          *string
	StartDate            *time.Time
	EndDate              *time.Time
	CreatedAt            time.Time
	CreatedBy            uuid.UUID
	UpdatedAt            time.Time
	UpdatedBy            *uuid.UUID
	DeletedAt            *time.Time
	DeletedBy            *uuid.UUID
}

// OwnerMemberSeed is the fully-resolved owner project_member that CreateWithOwner
// inserts in step 2 of the D40 3-step. It lives in `project` (NOT `projectmember`)
// so package `project` does not import package `projectmember` — that import + the
// projectmember→project owner-read edge would be a compile cycle. PK is pre-generated
// by the service. By construction ProjectRoleCode == project_owner literal and the
// row is active (no removed_at field here — the seed cannot represent a removed owner).
type OwnerMemberSeed struct {
	ID                    uuid.UUID
	WorkspaceID           uuid.UUID
	ProjectID             uuid.UUID
	WorkspaceMembershipID uuid.UUID
	ProjectRoleCode       string
	JoinedAt              time.Time
	CreatedBy             uuid.UUID
}
