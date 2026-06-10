package deliverabledbrepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	auditdbrepo "prasankit-api/internal/adapters/database/audit"
	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/deliverable"
	"prasankit-api/internal/modules/workspace"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DeliverableRepo implements deliverable.DeliverableRepository against Postgres via GORM.
//
// forbidSelfReview (config DELIVERABLE_FORBID_SELF_REVIEW) is held here, not in the service,
// because the self-review check needs the locked submission's submitted_by — only available
// under the deliverable FOR UPDATE lock (D64). Default false = self-review allowed.
type DeliverableRepo struct {
	db               *gorm.DB
	auditRepo        *auditdbrepo.AuditRepo
	forbidSelfReview bool
}

// NewDeliverableRepo constructs a DeliverableRepo.
func NewDeliverableRepo(db *gorm.DB, auditRepo *auditdbrepo.AuditRepo, forbidSelfReview bool) *DeliverableRepo {
	return &DeliverableRepo{db: db, auditRepo: auditRepo, forbidSelfReview: forbidSelfReview}
}

// ── deliverables CRUD ──────────────────────────────────────────────────────────

// CreateWithAudit atomically inserts a deliverable + audit_logs entry in one transaction.
// Composite FK violation (project not in ws) propagates as the raw wrapped error (DB backstop).
func (r *DeliverableRepo) CreateWithAudit(ctx context.Context, d deliverable.Deliverable, entry audit.Entry) error {
	model := deliverableToModel(d)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&model).Error; err != nil {
			return err
		}
		return r.auditRepo.LogTx(tx, entry)
	})
	if err != nil {
		return fmt.Errorf("create deliverable: %w", err)
	}
	return nil
}

// FindByIDForWorkspace returns the non-deleted deliverable scoped to workspace_id, or (nil, nil).
func (r *DeliverableRepo) FindByIDForWorkspace(ctx context.Context, workspaceID, deliverableID uuid.UUID) (*deliverable.Deliverable, error) {
	var m deliverableModel
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND id = ? AND deleted_at IS NULL", workspaceID, deliverableID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find deliverable: %w", err)
	}
	d := modelToDeliverable(m)
	return &d, nil
}

// UpdateWithAudit atomically updates a deliverable + audit_logs entry in one transaction.
func (r *DeliverableRepo) UpdateWithAudit(ctx context.Context, d deliverable.Deliverable, entry audit.Entry) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&deliverableModel{}).
			Where("workspace_id = ? AND id = ? AND deleted_at IS NULL", d.WorkspaceID, d.ID).
			Updates(map[string]any{
				"title":       d.Title,
				"description": d.Description,
				"due_date":    d.DueDate,
				"sort_order":  d.SortOrder,
				"updated_at":  d.UpdatedAt,
				"updated_by":  d.UpdatedBy,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return deliverable.ErrDeliverableNotFound
		}
		return r.auditRepo.LogTx(tx, entry)
	})
	if err != nil {
		if errors.Is(err, deliverable.ErrDeliverableNotFound) {
			return err
		}
		return fmt.Errorf("update deliverable: %w", err)
	}
	return nil
}

// SoftDeleteWithAudit atomically soft-deletes a deliverable + audit_logs entry.
func (r *DeliverableRepo) SoftDeleteWithAudit(ctx context.Context, workspaceID, deliverableID, deletedBy uuid.UUID, entry audit.Entry) error {
	now := time.Now().UTC()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&deliverableModel{}).
			Where("workspace_id = ? AND id = ? AND deleted_at IS NULL", workspaceID, deliverableID).
			Updates(map[string]any{
				"deleted_at": now,
				"deleted_by": deletedBy,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return deliverable.ErrDeliverableNotFound
		}
		return r.auditRepo.LogTx(tx, entry)
	})
	if err != nil {
		if errors.Is(err, deliverable.ErrDeliverableNotFound) {
			return err
		}
		return fmt.Errorf("soft delete deliverable: %w", err)
	}
	return nil
}

// ── derive read projections (LATERAL latest submission + its review) ─────────────

// latestRow is the flat row of a deliverable LEFT JOIN LATERAL (latest submission) LEFT JOIN
// (its review). Submission/review columns are NULL when absent.
type latestRow struct {
	// deliverable
	ID          uuid.UUID  `gorm:"column:id"`
	WorkspaceID uuid.UUID  `gorm:"column:workspace_id"`
	ProjectID   uuid.UUID  `gorm:"column:project_id"`
	Title       string     `gorm:"column:title"`
	Description *string    `gorm:"column:description"`
	DueDate     *time.Time `gorm:"column:due_date"`
	SortOrder   int        `gorm:"column:sort_order"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	CreatedBy   uuid.UUID  `gorm:"column:created_by"`
	UpdatedAt   time.Time  `gorm:"column:updated_at"`
	UpdatedBy   *uuid.UUID `gorm:"column:updated_by"`
	DeletedAt   *time.Time `gorm:"column:deleted_at"`
	DeletedBy   *uuid.UUID `gorm:"column:deleted_by"`
	// latest submission (nullable)
	SubID          *uuid.UUID `gorm:"column:sub_id"`
	SubRoundNo     *int       `gorm:"column:sub_round_no"`
	SubNote        *string    `gorm:"column:sub_note"`
	SubURL         *string    `gorm:"column:sub_url"`
	SubSubmittedBy *uuid.UUID `gorm:"column:sub_submitted_by"`
	SubSubmittedAt *time.Time `gorm:"column:sub_submitted_at"`
	SubCreatedAt   *time.Time `gorm:"column:sub_created_at"`
	SubCreatedBy   *uuid.UUID `gorm:"column:sub_created_by"`
	// review of that submission (nullable)
	RevID           *uuid.UUID `gorm:"column:rev_id"`
	RevDecisionCode *string    `gorm:"column:rev_decision_code"`
	RevComment      *string    `gorm:"column:rev_comment"`
	RevReviewedBy   *uuid.UUID `gorm:"column:rev_reviewed_by"`
	RevReviewedAt   *time.Time `gorm:"column:rev_reviewed_at"`
	RevCreatedAt    *time.Time `gorm:"column:rev_created_at"`
	RevCreatedBy    *uuid.UUID `gorm:"column:rev_created_by"`
}

// selectLatest is the column/JOIN spine shared by list + get. The LATERAL picks MAX(round_no)
// (round UNIQUE → 1 row), then LEFT JOIN its review (1/submission UNIQUE). project_id-bound
// derive driver per §M3.3.
const lateralJoin = `
LEFT JOIN LATERAL (
    SELECT s.id, s.round_no, s.note, s.url, s.submitted_by, s.submitted_at, s.created_at, s.created_by
    FROM submissions s
    WHERE s.workspace_id = d.workspace_id AND s.deliverable_id = d.id
    ORDER BY s.round_no DESC
    LIMIT 1
) sub ON true
LEFT JOIN submission_reviews rev
    ON rev.workspace_id = d.workspace_id AND rev.submission_id = sub.id`

const lateralSelect = `
d.id, d.workspace_id, d.project_id, d.title, d.description, d.due_date, d.sort_order,
d.created_at, d.created_by, d.updated_at, d.updated_by, d.deleted_at, d.deleted_by,
sub.id AS sub_id, sub.round_no AS sub_round_no, sub.note AS sub_note, sub.url AS sub_url,
sub.submitted_by AS sub_submitted_by, sub.submitted_at AS sub_submitted_at,
sub.created_at AS sub_created_at, sub.created_by AS sub_created_by,
rev.id AS rev_id, rev.decision_code AS rev_decision_code, rev.comment AS rev_comment,
rev.reviewed_by AS rev_reviewed_by, rev.reviewed_at AS rev_reviewed_at,
rev.created_at AS rev_created_at, rev.created_by AS rev_created_by`

func (row latestRow) toDeliverableWithStatus() deliverable.DeliverableWithStatus {
	d := deliverable.Deliverable{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		ProjectID:   row.ProjectID,
		Title:       row.Title,
		Description: row.Description,
		DueDate:     row.DueDate,
		SortOrder:   row.SortOrder,
		CreatedAt:   row.CreatedAt,
		CreatedBy:   row.CreatedBy,
		UpdatedAt:   row.UpdatedAt,
		UpdatedBy:   row.UpdatedBy,
		DeletedAt:   row.DeletedAt,
		DeletedBy:   row.DeletedBy,
	}
	out := deliverable.DeliverableWithStatus{Deliverable: d}
	if row.SubID != nil {
		sub := deliverable.Submission{
			ID:            *row.SubID,
			WorkspaceID:   row.WorkspaceID,
			ProjectID:     row.ProjectID,
			DeliverableID: row.ID,
			RoundNo:       deref(row.SubRoundNo),
			Note:          row.SubNote,
			URL:           row.SubURL,
			SubmittedBy:   derefUUID(row.SubSubmittedBy),
			SubmittedAt:   derefTime(row.SubSubmittedAt),
			CreatedAt:     derefTime(row.SubCreatedAt),
			CreatedBy:     derefUUID(row.SubCreatedBy),
		}
		swr := &deliverable.SubmissionWithReview{Submission: sub}
		if row.RevID != nil {
			swr.Review = &deliverable.SubmissionReview{
				ID:           *row.RevID,
				WorkspaceID:  row.WorkspaceID,
				ProjectID:    row.ProjectID,
				SubmissionID: *row.SubID,
				DecisionCode: derefStr(row.RevDecisionCode),
				Comment:      row.RevComment,
				ReviewedBy:   derefUUID(row.RevReviewedBy),
				ReviewedAt:   derefTime(row.RevReviewedAt),
				CreatedAt:    derefTime(row.RevCreatedAt),
				CreatedBy:    derefUUID(row.RevCreatedBy),
			}
		}
		out.LatestSubmission = swr
	}
	return out
}

// ListByProjectWithStatus returns all non-deleted deliverables of the project with their
// latest submission + review. Ordered sort_order ASC, created_at ASC tiebreak. No pagination.
func (r *DeliverableRepo) ListByProjectWithStatus(ctx context.Context, workspaceID, projectID uuid.UUID) ([]deliverable.DeliverableWithStatus, error) {
	var rows []latestRow
	err := r.db.WithContext(ctx).
		Table("deliverables AS d").
		Select(lateralSelect).
		Joins(lateralJoin).
		Where("d.workspace_id = ? AND d.project_id = ? AND d.deleted_at IS NULL", workspaceID, projectID).
		Order("d.sort_order ASC, d.created_at ASC").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list deliverables with status: %w", err)
	}
	out := make([]deliverable.DeliverableWithStatus, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.toDeliverableWithStatus())
	}
	return out, nil
}

// GetWithStatus returns one non-deleted deliverable + its latest submission/review, or (nil, nil).
func (r *DeliverableRepo) GetWithStatus(ctx context.Context, workspaceID, deliverableID uuid.UUID) (*deliverable.DeliverableWithStatus, error) {
	var row latestRow
	err := r.db.WithContext(ctx).
		Table("deliverables AS d").
		Select(lateralSelect).
		Joins(lateralJoin).
		Where("d.workspace_id = ? AND d.id = ? AND d.deleted_at IS NULL", workspaceID, deliverableID).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get deliverable with status: %w", err)
	}
	dws := row.toDeliverableWithStatus()
	return &dws, nil
}

// ListSubmissionsForDeliverable returns every submission (round DESC) of the deliverable with
// its review (LEFT JOIN, 1/submission). Used by the detail endpoint's submissions history.
func (r *DeliverableRepo) ListSubmissionsForDeliverable(ctx context.Context, workspaceID, deliverableID uuid.UUID) ([]deliverable.SubmissionWithReview, error) {
	type subRow struct {
		ID          uuid.UUID `gorm:"column:id"`
		ProjectID   uuid.UUID `gorm:"column:project_id"`
		RoundNo     int       `gorm:"column:round_no"`
		Note        *string   `gorm:"column:note"`
		URL         *string   `gorm:"column:url"`
		SubmittedBy uuid.UUID `gorm:"column:submitted_by"`
		SubmittedAt time.Time `gorm:"column:submitted_at"`
		CreatedAt   time.Time `gorm:"column:created_at"`
		CreatedBy   uuid.UUID `gorm:"column:created_by"`

		RevID           *uuid.UUID `gorm:"column:rev_id"`
		RevDecisionCode *string    `gorm:"column:rev_decision_code"`
		RevComment      *string    `gorm:"column:rev_comment"`
		RevReviewedBy   *uuid.UUID `gorm:"column:rev_reviewed_by"`
		RevReviewedAt   *time.Time `gorm:"column:rev_reviewed_at"`
		RevCreatedAt    *time.Time `gorm:"column:rev_created_at"`
		RevCreatedBy    *uuid.UUID `gorm:"column:rev_created_by"`
	}
	var rows []subRow
	err := r.db.WithContext(ctx).
		Table("submissions AS s").
		Select(`s.id, s.project_id, s.round_no, s.note, s.url, s.submitted_by, s.submitted_at,
			s.created_at, s.created_by,
			rev.id AS rev_id, rev.decision_code AS rev_decision_code, rev.comment AS rev_comment,
			rev.reviewed_by AS rev_reviewed_by, rev.reviewed_at AS rev_reviewed_at,
			rev.created_at AS rev_created_at, rev.created_by AS rev_created_by`).
		Joins(`LEFT JOIN submission_reviews rev ON rev.workspace_id = s.workspace_id AND rev.submission_id = s.id`).
		Where("s.workspace_id = ? AND s.deliverable_id = ?", workspaceID, deliverableID).
		Order("s.round_no DESC").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list submissions: %w", err)
	}
	out := make([]deliverable.SubmissionWithReview, 0, len(rows))
	for _, row := range rows {
		sub := deliverable.Submission{
			ID:            row.ID,
			WorkspaceID:   workspaceID,
			ProjectID:     row.ProjectID,
			DeliverableID: deliverableID,
			RoundNo:       row.RoundNo,
			Note:          row.Note,
			URL:           row.URL,
			SubmittedBy:   row.SubmittedBy,
			SubmittedAt:   row.SubmittedAt,
			CreatedAt:     row.CreatedAt,
			CreatedBy:     row.CreatedBy,
		}
		swr := deliverable.SubmissionWithReview{Submission: sub}
		if row.RevID != nil {
			swr.Review = &deliverable.SubmissionReview{
				ID:           *row.RevID,
				WorkspaceID:  workspaceID,
				ProjectID:    row.ProjectID,
				SubmissionID: row.ID,
				DecisionCode: derefStr(row.RevDecisionCode),
				Comment:      row.RevComment,
				ReviewedBy:   derefUUID(row.RevReviewedBy),
				ReviewedAt:   derefTime(row.RevReviewedAt),
				CreatedAt:    derefTime(row.RevCreatedAt),
				CreatedBy:    derefUUID(row.RevCreatedBy),
			}
		}
		out = append(out, swr)
	}
	return out, nil
}

// ── submit (D64 — deliverable FOR UPDATE as first statement of the tx) ───────────

// CreateSubmissionLocked runs the mandatory serialization sequence: SELECT ... FOR UPDATE the
// deliverable row → MAX(round_no) + latest review under the lock → accept-terminal guard →
// INSERT round_no=MAX+1 → audit. UNIQUE(ws, deliverable_id, round_no) fire after the lock = bug
// → propagate (fail loud, D64; NO retry).
func (r *DeliverableRepo) CreateSubmissionLocked(ctx context.Context, s deliverable.Submission, entry audit.Entry) (*deliverable.Submission, error) {
	var created deliverable.Submission
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Statement 1 (D64): lock the deliverable row (serialization point). Scan into a struct
		// (GORM maps the uuid column correctly; a bare uuid.UUID Scan fails on the pq driver).
		var locked struct {
			ID uuid.UUID `gorm:"column:id"`
		}
		lockErr := tx.Model(&deliverableModel{}).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id").
			Where("workspace_id = ? AND id = ? AND deleted_at IS NULL", s.WorkspaceID, s.DeliverableID).
			Limit(1).
			Scan(&locked).Error
		if lockErr != nil {
			return lockErr
		}
		if locked.ID == uuid.Nil {
			return deliverable.ErrDeliverableNotFound
		}

		// Under the lock: latest round + its review.
		var latest submissionModel
		latestErr := tx.
			Where("workspace_id = ? AND deliverable_id = ?", s.WorkspaceID, s.DeliverableID).
			Order("round_no DESC").
			Limit(1).
			First(&latest).Error
		nextRound := 1
		if latestErr == nil {
			// A previous round exists — accept-terminal guard (D58/OQ-7).
			var rev reviewModel
			revErr := tx.
				Where("workspace_id = ? AND submission_id = ?", s.WorkspaceID, latest.ID).
				First(&rev).Error
			if revErr == nil && rev.DecisionCode == deliverable.SubmissionDecisionAccepted {
				return deliverable.ErrAcceptTerminal
			}
			if revErr != nil && !errors.Is(revErr, gorm.ErrRecordNotFound) {
				return revErr
			}
			nextRound = latest.RoundNo + 1
		} else if !errors.Is(latestErr, gorm.ErrRecordNotFound) {
			return latestErr
		}

		s.RoundNo = nextRound
		model := submissionToModel(s)
		if err := tx.Create(&model).Error; err != nil {
			// UNIQUE(ws, deliverable_id, round_no) here = lock lost = bug → fail loud (D64).
			return err
		}
		created = modelToSubmission(model)

		// Re-stamp round_no into the audit NewValue snapshot (placeholder was 0 at service).
		if entry.NewValue != nil {
			entry.NewValue["round_no"] = nextRound
		}
		return r.auditRepo.LogTx(tx, entry)
	})
	if err != nil {
		if errors.Is(err, deliverable.ErrDeliverableNotFound) || errors.Is(err, deliverable.ErrAcceptTerminal) {
			return nil, err
		}
		return nil, fmt.Errorf("create submission: %w", err)
	}
	return &created, nil
}

// ── review (D64 — deliverable FOR UPDATE, SAME lock order as submit) ─────────────

// CreateReviewLocked: load submission (must belong to deliverableID) → FOR UPDATE its
// deliverable row (same lock order as submit) → re-check latest-only + already-reviewed +
// self-review under the lock → INSERT review + audit. UNIQUE(ws, submission_id) fire after the
// lock = bug → propagate (fail loud, D64; NO retry).
func (r *DeliverableRepo) CreateReviewLocked(ctx context.Context, deliverableID uuid.UUID, rv deliverable.SubmissionReview, entry audit.Entry) (*deliverable.SubmissionReview, error) {
	var created deliverable.SubmissionReview
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Load the submission (ws+id); it must belong to the URL deliverable.
		var sub submissionModel
		subErr := tx.
			Where("workspace_id = ? AND id = ?", rv.WorkspaceID, rv.SubmissionID).
			First(&sub).Error
		if subErr != nil {
			if errors.Is(subErr, gorm.ErrRecordNotFound) {
				return deliverable.ErrSubmissionNotFound
			}
			return subErr
		}
		if sub.DeliverableID != deliverableID {
			return deliverable.ErrSubmissionNotFound
		}

		// Statement: lock the submission's deliverable row (same order as submit → no deadlock).
		var locked struct {
			ID uuid.UUID `gorm:"column:id"`
		}
		lockErr := tx.Model(&deliverableModel{}).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id").
			Where("workspace_id = ? AND id = ? AND deleted_at IS NULL", rv.WorkspaceID, sub.DeliverableID).
			Limit(1).
			Scan(&locked).Error
		if lockErr != nil {
			return lockErr
		}
		if locked.ID == uuid.Nil {
			// Deliverable was deleted between submission load and lock → treat as not found.
			return deliverable.ErrSubmissionNotFound
		}

		// (a) Under the lock: submission must still be the MAX round.
		var maxRound int
		if err := tx.Model(&submissionModel{}).
			Where("workspace_id = ? AND deliverable_id = ?", rv.WorkspaceID, sub.DeliverableID).
			Select("COALESCE(MAX(round_no), 0)").
			Scan(&maxRound).Error; err != nil {
			return err
		}
		if sub.RoundNo != maxRound {
			return deliverable.ErrSubmissionSuperseded
		}

		// (b) Under the lock: submission must have no review yet.
		var existingRev int64
		if err := tx.Model(&reviewModel{}).
			Where("workspace_id = ? AND submission_id = ?", rv.WorkspaceID, rv.SubmissionID).
			Count(&existingRev).Error; err != nil {
			return err
		}
		if existingRev > 0 {
			return deliverable.ErrAlreadyReviewed
		}

		// (d) Self-review guard (OQ-6) — only when flag enabled. submitted_by known under lock.
		if r.forbidSelfReview && sub.SubmittedBy == rv.ReviewedBy {
			return deliverable.ErrSelfReviewForbidden
		}

		model := reviewToModel(rv)
		if err := tx.Create(&model).Error; err != nil {
			// UNIQUE(ws, submission_id) here = lock lost = bug → fail loud (D64).
			return err
		}
		created = modelToReview(model)
		return r.auditRepo.LogTx(tx, entry)
	})
	if err != nil {
		switch {
		case errors.Is(err, deliverable.ErrSubmissionNotFound),
			errors.Is(err, deliverable.ErrSubmissionSuperseded),
			errors.Is(err, deliverable.ErrAlreadyReviewed),
			errors.Is(err, deliverable.ErrSelfReviewForbidden):
			return nil, err
		}
		return nil, fmt.Errorf("create review: %w", err)
	}
	return &created, nil
}

// ── MasterRepo (submission_decisions pre-check) ──────────────────────────────────

// MasterRepo implements deliverable.MasterRepository against Postgres via GORM.
type MasterRepo struct {
	db *gorm.DB
}

// NewMasterRepo constructs a MasterRepo.
func NewMasterRepo(db *gorm.DB) *MasterRepo { return &MasterRepo{db: db} }

// IsActiveSubmissionDecisionCode returns true when code exists in submission_decisions active.
func (r *MasterRepo) IsActiveSubmissionDecisionCode(ctx context.Context, code string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).
		Table("submission_decisions").
		Where("code = ? AND status = 'active'", code).
		Limit(1).
		Count(&n).Error
	if err != nil {
		return false, fmt.Errorf("check decision_code: %w", err)
	}
	return n > 0, nil
}

// ── ProjectAccessRepo (project existence + actor's project_member resolution) ────

// ProjectAccessRepo implements deliverable.ProjectAccessRepository against Postgres via GORM.
type ProjectAccessRepo struct {
	db *gorm.DB
}

// NewProjectAccessRepo constructs a ProjectAccessRepo.
func NewProjectAccessRepo(db *gorm.DB) *ProjectAccessRepo { return &ProjectAccessRepo{db: db} }

// ProjectExistsForWorkspace returns true when a non-deleted project exists in this ws.
func (r *ProjectAccessRepo) ProjectExistsForWorkspace(ctx context.Context, workspaceID, projectID uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).
		Table("projects").
		Where("workspace_id = ? AND id = ? AND deleted_at IS NULL", workspaceID, projectID).
		Limit(1).
		Count(&n).Error
	if err != nil {
		return false, fmt.Errorf("check project exists: %w", err)
	}
	return n > 0, nil
}

// FindActiveMemberByMembership resolves the actor's active project_member row, joined to
// workspace_memberships to enforce the ws-active filter (a ws-suspended member is not active).
// (workspace_id, project_id, workspace_membership_id) + removed_at IS NULL + wm active.
// Returns (nil, nil) when not an active member.
func (r *ProjectAccessRepo) FindActiveMemberByMembership(ctx context.Context, workspaceID, projectID, membershipID uuid.UUID) (*deliverable.ActorMember, error) {
	type row struct {
		ID              uuid.UUID `gorm:"column:id"`
		ProjectRoleCode string    `gorm:"column:project_role_code"`
	}
	var m row
	err := r.db.WithContext(ctx).
		Table("project_members AS pm").
		Select("pm.id, pm.project_role_code").
		Joins("INNER JOIN workspace_memberships wm ON wm.workspace_id = pm.workspace_id AND wm.id = pm.workspace_membership_id").
		Where("pm.workspace_id = ? AND pm.project_id = ? AND pm.workspace_membership_id = ? AND pm.removed_at IS NULL AND wm.membership_status_code = ?",
			workspaceID, projectID, membershipID, workspace.MembershipStatusActive).
		Limit(1).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find active project_member: %w", err)
	}
	return &deliverable.ActorMember{ID: m.ID, ProjectRoleCode: m.ProjectRoleCode}, nil
}

// ── small deref helpers (LATERAL nullable columns) ───────────────────────────────

func deref(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
func derefUUID(p *uuid.UUID) uuid.UUID {
	if p == nil {
		return uuid.Nil
	}
	return *p
}
func derefTime(p *time.Time) time.Time {
	if p == nil {
		return time.Time{}
	}
	return *p
}
