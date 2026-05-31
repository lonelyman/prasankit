// Package projectposition contains the domain and service layer for the project_positions
// customizable master (D39 + §2.9). Domain structs and interfaces are pure — no gorm tags,
// no fiber imports.
package projectposition

import (
	"time"

	"github.com/google/uuid"
)

// ProjectPosition is the pure domain representation of a project_positions row.
// status='active'|'deprecated' REPLACES soft-delete (§2.9) — there is NO deleted_at.
type ProjectPosition struct {
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
