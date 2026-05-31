// Package projectmemberposition contains the domain and service layer for the
// project_member_positions M:N junction (D39 append/delete-only). Domain structs and
// interfaces are pure — no gorm tags, no fiber imports.
package projectmemberposition

import (
	"time"

	"github.com/google/uuid"
)

// ProjectMemberPosition is the pure domain representation of a project_member_positions row.
// APPEND/DELETE-ONLY (D39): no updated_at/updated_by/status/removed_at.
type ProjectMemberPosition struct {
	ID                  uuid.UUID
	WorkspaceID         uuid.UUID
	ProjectMemberID     uuid.UUID
	ProjectPositionCode string // raw string FK-by-code — NOT a projectposition.ProjectPosition (no Go-type coupling)
	CreatedAt           time.Time
	CreatedBy           uuid.UUID
}

// PositionWithLabel is the read projection for ListByMember (JOIN to project_positions).
// The labels are plain fields populated by the SQL JOIN — not a projectposition import.
type PositionWithLabel struct {
	ProjectMemberPosition
	LabelTH string
	LabelEN string
	Status  string
}
