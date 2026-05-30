package project

import "errors"

// Sentinel errors returned by the service and caught by the handler for HTTP mapping.
var (
	ErrProjectNotFound   = errors.New("project: not found")
	ErrSlugTaken         = errors.New("project: slug already taken")
	ErrInvalidStatusCode = errors.New("project: invalid project_status_code")
	ErrInvalidTypeCode   = errors.New("project: invalid project_type_code")
)

// FieldError describes a single validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError carries per-field validation details.
// Duplicate shape of workspace.ValidationError — each module owns its own validation
// surface, matching M1 pattern.
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string { return "project: validation failed" }
