package projectmemberposition

import "errors"

// Sentinel errors returned by the service and caught by the handler for HTTP mapping.
var (
	ErrAssignmentNotFound    = errors.New("project_member_position: assignment not found")          // 404 (unassign)
	ErrAlreadyAssigned       = errors.New("project_member_position: position already assigned")     // 409 (uq violation)
	ErrProjectMemberNotFound = errors.New("project_member_position: project member not found")      // 404 (member absent in this project)
	ErrMemberRemoved         = errors.New("project_member_position: project member is removed")     // 422 (removed_at IS NOT NULL) — §M2.5(c)
	ErrInvalidPositionCode   = errors.New("project_member_position: invalid project_position_code") // 422 (FK / not active)
	ErrProjectNotFound       = errors.New("project_member_position: project not found")             // 404 (D42 collapse)
)

// FieldError describes a single validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError carries per-field validation details.
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string { return "project_member_position: validation failed" }
