package workspace

import "errors"

// Sentinel errors returned by the service and caught by the handler for HTTP mapping.
var (
	ErrSlugReserved      = errors.New("workspace: slug is reserved")
	ErrSlugTaken         = errors.New("workspace: slug already taken")
	ErrWorkspaceNotFound = errors.New("workspace: not found")
	ErrForbidden         = errors.New("workspace: forbidden")

	// Invitation errors.
	ErrAlreadyMember        = errors.New("workspace: account is already an active member")
	ErrAlreadyPending       = errors.New("workspace: a pending invitation already exists for this email")
	ErrInvitationInvalid    = errors.New("workspace: invitation not found or invalid")
	ErrInvitationExpired    = errors.New("workspace: invitation has expired or is no longer active")
	ErrEmailMismatch        = errors.New("workspace: invitation email does not match authenticated account")
	ErrInviteRoleNotAllowed = errors.New("workspace: org role not allowed for invitation")
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
