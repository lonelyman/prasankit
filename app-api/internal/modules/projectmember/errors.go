package projectmember

import "errors"

// Sentinel errors returned by the service and caught by the handler for HTTP mapping.
var (
	ErrProjectMemberNotFound = errors.New("project_member: not found")
	ErrProjectNotFound       = errors.New("project_member: project not found") // project gate (404 collapse)
	ErrMembershipNotEligible = errors.New("project_member: target workspace_membership is not an active member of this workspace")
	ErrAlreadyMember         = errors.New("project_member: membership already an active member of this project")
	ErrInvalidRoleCode       = errors.New("project_member: invalid project_role_code")
	ErrOwnerRoleImmutable    = errors.New("project_member: cannot change role/remove the project owner outside ownership transfer")
)

// FieldError describes a single validation failure.
// Duplicate shape of project.FieldError — each module owns its own validation surface.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError carries per-field validation details.
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string { return "project_member: validation failed" }
