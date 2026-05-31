package companyposition

import "errors"

// Sentinel errors returned by the service and caught by the handler for HTTP mapping.
var (
	ErrPositionNotFound = errors.New("company_position: not found")                                  // 404
	ErrCodeTaken        = errors.New("company_position: code already exists in this workspace")      // 409
	ErrCodeImmutable    = errors.New("company_position: code is immutable")                          // 422
	ErrSystemImmutable  = errors.New("company_position: system position cannot be modified")         // 422
	ErrInvalidStatus    = errors.New("company_position: invalid status (must be active|deprecated)") // 422

	// ws company_position attach errors.
	ErrInvalidPositionCode = errors.New("company_position: invalid company_position_code")    // 422
	ErrMembershipNotFound  = errors.New("company_position: membership not found or inactive") // 404
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

func (e *ValidationError) Error() string { return "company_position: validation failed" }
