package deliverable

import (
	"context"

	"prasankit-api/internal/modules/audit"

	"github.com/google/uuid"
)

// DeliverableRepository defines storage operations on deliverables / submissions /
// submission_reviews. All ws-scoped methods take workspace_id and put it in WHERE
// (isolation invariant). Mutations write an audit_logs row in the SAME transaction (D31).
type DeliverableRepository interface {
	// ── deliverables (soft-delete CRUD) ─────────────────────────────────────────
	CreateWithAudit(ctx context.Context, d Deliverable, entry audit.Entry) error
	// FindByIDForWorkspace returns the non-deleted deliverable, or (nil, nil) if absent.
	FindByIDForWorkspace(ctx context.Context, workspaceID, deliverableID uuid.UUID) (*Deliverable, error)
	// ListByProjectWithStatus returns all non-deleted deliverables of the project with their
	// latest submission + that submission's review (LATERAL derive query). Ordered
	// sort_order ASC, created_at ASC tiebreak. No pagination (bounded per project).
	ListByProjectWithStatus(ctx context.Context, workspaceID, projectID uuid.UUID) ([]DeliverableWithStatus, error)
	// GetWithStatus returns one non-deleted deliverable + its latest submission/review, or
	// (nil, nil) if absent.
	GetWithStatus(ctx context.Context, workspaceID, deliverableID uuid.UUID) (*DeliverableWithStatus, error)
	UpdateWithAudit(ctx context.Context, d Deliverable, entry audit.Entry) error
	SoftDeleteWithAudit(ctx context.Context, workspaceID, deliverableID, deletedBy uuid.UUID, entry audit.Entry) error

	// ── submissions history ─────────────────────────────────────────────────────
	// ListSubmissionsForDeliverable returns every submission (round DESC) with its review.
	ListSubmissionsForDeliverable(ctx context.Context, workspaceID, deliverableID uuid.UUID) ([]SubmissionWithReview, error)

	// ── submit (D64 — deliverable FOR UPDATE as first statement) ─────────────────
	// CreateSubmissionLocked runs the mandatory serialization sequence in ONE tx:
	//   1. SELECT id FROM deliverables WHERE ws=? AND id=? AND deleted_at IS NULL FOR UPDATE
	//      → not found ⇒ ErrDeliverableNotFound
	//   2. find MAX(round_no) + its review under the lock
	//      → latest review = accepted ⇒ ErrAcceptTerminal (D58/OQ-7)
	//   3. INSERT submission round_no = MAX+1 (1 when none), submitted_at = now()
	//   4. LogTx(audit submission.create, bind project_id)
	// The caller pre-resolved submittedBy (actor's project_member id) — never from body.
	// UNIQUE(ws, deliverable_id, round_no) fire after the lock = bug → propagate (fail loud, D64).
	CreateSubmissionLocked(ctx context.Context, s Submission, entry audit.Entry) (*Submission, error)

	// ── review (D64 — deliverable FOR UPDATE, same lock order as submit) ─────────
	// CreateReviewLocked runs in ONE tx:
	//   1. load submission (ws+id); must belong to deliverableID ⇒ else ErrSubmissionNotFound
	//   2. SELECT ... FOR UPDATE the submission's deliverable row (same lock order as submit)
	//   3. under lock: submission must still be MAX round ⇒ else ErrSubmissionSuperseded
	//   4. under lock: submission must have no review ⇒ else ErrAlreadyReviewed
	//   5. INSERT review + LogTx(audit submission_review.create, bind project_id)
	// decision_code active-check + self-review check happen in the service BEFORE this call.
	// reviewedBy pre-resolved (actor's project_member id). UNIQUE(ws, submission_id) fire after
	// the lock = bug → propagate (fail loud, D64; NO retry).
	CreateReviewLocked(ctx context.Context, deliverableID uuid.UUID, r SubmissionReview, entry audit.Entry) (*SubmissionReview, error)
}

// MasterRepository pre-checks master vocabulary by code (D28).
type MasterRepository interface {
	IsActiveSubmissionDecisionCode(ctx context.Context, code string) (bool, error)
}

// ProjectAccessRepository resolves project existence + the actor's active project_member
// (for the D63 service-layer authz gate). It reads project_members / projects directly —
// the deliverable module does NOT import the projectmember/project modules (each module
// owns its own read here, mirroring how projectmember inlines its project read).
type ProjectAccessRepository interface {
	// ProjectExistsForWorkspace returns true when a non-deleted project exists in this ws.
	ProjectExistsForWorkspace(ctx context.Context, workspaceID, projectID uuid.UUID) (bool, error)
	// FindActiveMemberByMembership resolves the actor's active project_member row for
	// (workspace_id, project_id, workspace_membership_id) with removed_at IS NULL.
	// Returns (nil, nil) when the actor is not an active member of the project.
	FindActiveMemberByMembership(ctx context.Context, workspaceID, projectID, membershipID uuid.UUID) (*ActorMember, error)
}

// ActorMember is the minimal projection of the actor's project_member row used by authz +
// submitted_by/reviewed_by derivation. Defined here (not imported from projectmember) to
// keep the module dependency-free.
type ActorMember struct {
	ID              uuid.UUID // project_members.id — the FK target for submitted_by/reviewed_by
	ProjectRoleCode string
}
