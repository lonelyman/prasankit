package deliverable

// Submission decision codes (FK-by-code → submission_decisions; D58 — rename = breaking,
// derive branches on these literals). MUST string-equal the seed in migration 000018.
const (
	SubmissionDecisionAccepted    = "accepted"
	SubmissionDecisionConditional = "conditional"
	SubmissionDecisionRejected    = "rejected"
)

// Derived deliverable status codes (label dictionary deliverable_statuses; D59 — NOT a
// stored column, computed by DeriveStatus). MUST string-equal the seed in migration 000019.
// The three terminal-verdict codes (Accepted/Conditional/Rejected) MUST string-equal the
// submission-decision codes above — cross-master drift guard lives in derive_test.go.
const (
	DeliverableStatusNotSubmitted = "not_submitted"
	DeliverableStatusInReview     = "in_review"
	DeliverableStatusAccepted     = "accepted"
	DeliverableStatusConditional  = "conditional"
	DeliverableStatusRejected     = "rejected"
)

// Timeliness codes (derived at presenter from due_date vs submitted_at; D62 — no column/master).
const (
	TimelinessNoDue  = "no_due"
	TimelinessEarly  = "early"
	TimelinessOnTime = "on_time"
	TimelinessLate   = "late"
)

// Project role codes (D63 authz gate). These literals are owned by the projectmember /
// project master vocabulary; inlined here (module owns its own contract surface, same as
// project/projectmember inline literals to avoid an import cycle — projectmember imports
// nothing from deliverable, but deliverable resolving the actor's role does not need that
// package). MUST string-equal the seed in migration 000009.
const (
	ProjectRoleOwner   = "project_owner"
	ProjectRoleManager = "project_manager"
	ProjectRoleViewer  = "viewer"
)

// Audit action codes (D29 — Go consts, no FK, no DB CHECK on vocabulary).
const (
	AuditActionDeliverableCreate = "deliverable.create"
	AuditActionDeliverableUpdate = "deliverable.update"
	AuditActionDeliverableDelete = "deliverable.delete"
	AuditActionSubmissionCreate  = "submission.create"
	AuditActionReviewCreate      = "submission_review.create"

	AuditResourceTypeDeliverable = "deliverable"
	AuditResourceTypeSubmission  = "submission"
	AuditResourceTypeReview      = "submission_review"

	// AuditResultSuccess is a local copy of the success literal (each module owns its own
	// audit vocabulary). MUST equal "success" to match existing audit_logs rows.
	AuditResultSuccess = "success"
)
