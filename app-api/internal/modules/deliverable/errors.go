package deliverable

import "errors"

// Sentinel errors returned by the service and caught by the handler for HTTP mapping.
// The handler maps each to a stable {status, code} pair (see deliverable_handler.go).
var (
	// 404 probe-collapse (D42) — deliverable absent / project not in ws / soft-deleted /
	// non-member read. NEVER leak existence.
	ErrDeliverableNotFound = errors.New("deliverable: deliverable not found")
	// 404 — project absent / not in ws / soft-deleted (resource-typed probe-collapse).
	ErrProjectNotFound = errors.New("deliverable: project not found")
	// 404 — submission absent / not belonging to the URL deliverable.
	ErrSubmissionNotFound = errors.New("deliverable: submission not found")

	// 403-equivalent — actor lacks the project-role / org-role required for the action.
	ErrForbidden = errors.New("deliverable: forbidden")

	// 422 — org Owner/Admin bypass cannot submit/review without a project_member row
	// (FK requires one). deliverable.not_project_member.
	ErrNotProjectMember = errors.New("deliverable: actor is not an active project member")

	// 409 — accept is terminal (D58/OQ-7): resubmit after accepted is blocked at submit-time.
	ErrAcceptTerminal = errors.New("deliverable: latest submission already accepted (terminal)")
	// 409 — reviewed submission is no longer the latest round (a newer submission exists).
	ErrSubmissionSuperseded = errors.New("deliverable: submission superseded by a newer round")
	// 409 — submission already has a review (verdict immutable; 1 review/submission).
	ErrAlreadyReviewed = errors.New("deliverable: submission already reviewed")

	// 422 — decision_code is not an active row in submission_decisions master.
	ErrInvalidMasterCode = errors.New("deliverable: invalid decision_code")
	// 422 — ForbidSelfReview=true and reviewer == submitter (OQ-6 separation-of-duty).
	ErrSelfReviewForbidden = errors.New("deliverable: self-review forbidden")
)

// FieldError describes a single validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError carries per-field validation details (each module owns its own
// validation surface, matching the project/workspace pattern).
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string { return "deliverable: validation failed" }
