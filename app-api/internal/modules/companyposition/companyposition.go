// Package companyposition contains the domain and service layer for the company_positions
// customizable master (D39 + D41 + §2.9), plus the workspace_memberships.company_position_code
// attach (D41). Domain structs and interfaces are pure — no gorm tags, no fiber imports.
package companyposition

import (
	"time"

	"github.com/google/uuid"
)

// CompanyPosition is the pure domain representation of a company_positions row.
// status='active'|'deprecated' REPLACES soft-delete (§2.9) — there is NO deleted_at.
type CompanyPosition struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Code        string
	LabelTH     string
	LabelEN     string
	Description *string
	SortOrder   int
	IsSystem    bool
	Status      string // "active" | "deprecated"
	CreatedAt   time.Time
	CreatedBy   uuid.UUID
	UpdatedAt   time.Time
	UpdatedBy   *uuid.UUID
}
