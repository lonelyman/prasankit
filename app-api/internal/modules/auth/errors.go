package auth

import "errors"

// Sentinel errors returned by service and caught by handler for HTTP mapping.
var (
	ErrInvalidInput       = errors.New("auth: invalid input")
	ErrEmailTaken         = errors.New("auth: email taken")
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
	ErrAccountLocked      = errors.New("auth: account locked")
	ErrUnauthenticated    = errors.New("auth: unauthenticated")
	ErrEmailNotVerified   = errors.New("auth: email not verified")
	ErrTokenInvalid       = errors.New("auth: token invalid")
	ErrTokenExpired       = errors.New("auth: token expired")
)

// ValidationError carries per-field validation details.
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	return ErrInvalidInput.Error()
}

// FieldError describes a single validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}
