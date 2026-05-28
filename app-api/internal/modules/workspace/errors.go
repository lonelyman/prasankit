package workspace

import "errors"

// Sentinel errors returned by the service and caught by the handler for HTTP mapping.
var (
	ErrSlugReserved      = errors.New("workspace: slug is reserved")
	ErrSlugTaken         = errors.New("workspace: slug already taken")
	ErrWorkspaceNotFound = errors.New("workspace: not found")
	ErrForbidden         = errors.New("workspace: forbidden")
)

// ValidationError carries per-field validation details (same pattern as auth).
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	return "workspace: invalid input"
}

// FieldError describes a single validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}
