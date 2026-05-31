package projectposition

import "errors"

// Sentinel errors returned by the service and caught by the handler for HTTP mapping.
var (
	ErrPositionNotFound = errors.New("project_position: not found")                                  // 404
	ErrCodeTaken        = errors.New("project_position: code already exists in this workspace")      // 409
	ErrCodeImmutable    = errors.New("project_position: code is immutable")                          // 422
	ErrSystemImmutable  = errors.New("project_position: system position cannot be modified")         // 422
	ErrInvalidStatus    = errors.New("project_position: invalid status (must be active|deprecated)") // 422
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

func (e *ValidationError) Error() string { return "project_position: validation failed" }
