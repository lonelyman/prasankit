// Package deliverable contains the domain and service layer for the deliverables /
// submissions / submission_reviews entities (M3 — งวดงาน + ส่งหลายรอบ + ตรวจรับ).
// Domain structs and interfaces are pure — no gorm tags, no fiber imports.
package deliverable

import (
	"time"

	"github.com/google/uuid"
)

// Deliverable is the pure domain representation of a deliverables row (งวดงาน, soft-delete).
// No gorm tags — mapping happens in adapters/database/deliverable.
type Deliverable struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	Title       string
	Description *string
	DueDate     *time.Time // DATE; nil = no due date
	SortOrder   int
	CreatedAt   time.Time
	CreatedBy   uuid.UUID
	UpdatedAt   time.Time
	UpdatedBy   *uuid.UUID
	DeletedAt   *time.Time
	DeletedBy   *uuid.UUID
}

// Submission is the pure domain representation of a submissions row (ส่งจริงหลายรอบ, append-only).
type Submission struct {
	ID            uuid.UUID
	WorkspaceID   uuid.UUID
	ProjectID     uuid.UUID
	DeliverableID uuid.UUID
	RoundNo       int
	Note          *string
	URL           *string
	SubmittedBy   uuid.UUID // project_members.id (D60)
	SubmittedAt   time.Time
	CreatedAt     time.Time
	CreatedBy     uuid.UUID
}

// SubmissionReview is the pure domain representation of a submission_reviews row
// (ตรวจรับ, append-only, 1 per submission, verdict immutable).
type SubmissionReview struct {
	ID           uuid.UUID
	WorkspaceID  uuid.UUID
	ProjectID    uuid.UUID
	SubmissionID uuid.UUID
	DecisionCode string
	Comment      *string
	ReviewedBy   uuid.UUID // project_members.id (D60)
	ReviewedAt   time.Time
	CreatedAt    time.Time
	CreatedBy    uuid.UUID
}

// SubmissionWithReview is the read projection of a submission + its (optional) review.
// Built from the LATERAL derive query; Review is nil when the submission has no review.
type SubmissionWithReview struct {
	Submission
	Review *SubmissionReview // nil = no review yet (→ in_review when this is the latest round)
}

// DeliverableWithStatus is the read projection of a deliverable + its latest submission
// (with that submission's review, if any). LatestSubmission is nil when not_submitted.
// The derived status_code is computed by derive.go (DeriveStatus) at the presenter — it is
// NOT stored (D59). This struct carries only the raw facts the derive funcs consume.
type DeliverableWithStatus struct {
	Deliverable
	LatestSubmission *SubmissionWithReview // nil = no submission (not_submitted)
}
