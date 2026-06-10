// Package deliverabledbrepo implements the deliverable domain ports against Postgres via GORM.
package deliverabledbrepo

import (
	"time"

	"prasankit-api/internal/modules/deliverable"

	"github.com/google/uuid"
)

// deliverableModel is the GORM model for deliverables.
type deliverableModel struct {
	ID          uuid.UUID  `gorm:"column:id;primaryKey"`
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
}

func (deliverableModel) TableName() string { return "deliverables" }

// submissionModel is the GORM model for submissions (append-only).
type submissionModel struct {
	ID            uuid.UUID `gorm:"column:id;primaryKey"`
	WorkspaceID   uuid.UUID `gorm:"column:workspace_id"`
	ProjectID     uuid.UUID `gorm:"column:project_id"`
	DeliverableID uuid.UUID `gorm:"column:deliverable_id"`
	RoundNo       int       `gorm:"column:round_no"`
	Note          *string   `gorm:"column:note"`
	URL           *string   `gorm:"column:url"`
	SubmittedBy   uuid.UUID `gorm:"column:submitted_by"`
	SubmittedAt   time.Time `gorm:"column:submitted_at"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	CreatedBy     uuid.UUID `gorm:"column:created_by"`
}

func (submissionModel) TableName() string { return "submissions" }

// reviewModel is the GORM model for submission_reviews (append-only).
type reviewModel struct {
	ID           uuid.UUID `gorm:"column:id;primaryKey"`
	WorkspaceID  uuid.UUID `gorm:"column:workspace_id"`
	ProjectID    uuid.UUID `gorm:"column:project_id"`
	SubmissionID uuid.UUID `gorm:"column:submission_id"`
	DecisionCode string    `gorm:"column:decision_code"`
	Comment      *string   `gorm:"column:comment"`
	ReviewedBy   uuid.UUID `gorm:"column:reviewed_by"`
	ReviewedAt   time.Time `gorm:"column:reviewed_at"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	CreatedBy    uuid.UUID `gorm:"column:created_by"`
}

func (reviewModel) TableName() string { return "submission_reviews" }

// ── mappers ──────────────────────────────────────────────────────────────────────

func deliverableToModel(d deliverable.Deliverable) deliverableModel {
	return deliverableModel{
		ID:          d.ID,
		WorkspaceID: d.WorkspaceID,
		ProjectID:   d.ProjectID,
		Title:       d.Title,
		Description: d.Description,
		DueDate:     d.DueDate,
		SortOrder:   d.SortOrder,
		CreatedAt:   d.CreatedAt,
		CreatedBy:   d.CreatedBy,
		UpdatedAt:   d.UpdatedAt,
		UpdatedBy:   d.UpdatedBy,
		DeletedAt:   d.DeletedAt,
		DeletedBy:   d.DeletedBy,
	}
}

func modelToDeliverable(m deliverableModel) deliverable.Deliverable {
	return deliverable.Deliverable{
		ID:          m.ID,
		WorkspaceID: m.WorkspaceID,
		ProjectID:   m.ProjectID,
		Title:       m.Title,
		Description: m.Description,
		DueDate:     m.DueDate,
		SortOrder:   m.SortOrder,
		CreatedAt:   m.CreatedAt,
		CreatedBy:   m.CreatedBy,
		UpdatedAt:   m.UpdatedAt,
		UpdatedBy:   m.UpdatedBy,
		DeletedAt:   m.DeletedAt,
		DeletedBy:   m.DeletedBy,
	}
}

func submissionToModel(s deliverable.Submission) submissionModel {
	return submissionModel{
		ID:            s.ID,
		WorkspaceID:   s.WorkspaceID,
		ProjectID:     s.ProjectID,
		DeliverableID: s.DeliverableID,
		RoundNo:       s.RoundNo,
		Note:          s.Note,
		URL:           s.URL,
		SubmittedBy:   s.SubmittedBy,
		SubmittedAt:   s.SubmittedAt,
		CreatedAt:     s.CreatedAt,
		CreatedBy:     s.CreatedBy,
	}
}

func modelToSubmission(m submissionModel) deliverable.Submission {
	return deliverable.Submission{
		ID:            m.ID,
		WorkspaceID:   m.WorkspaceID,
		ProjectID:     m.ProjectID,
		DeliverableID: m.DeliverableID,
		RoundNo:       m.RoundNo,
		Note:          m.Note,
		URL:           m.URL,
		SubmittedBy:   m.SubmittedBy,
		SubmittedAt:   m.SubmittedAt,
		CreatedAt:     m.CreatedAt,
		CreatedBy:     m.CreatedBy,
	}
}

func reviewToModel(r deliverable.SubmissionReview) reviewModel {
	return reviewModel{
		ID:           r.ID,
		WorkspaceID:  r.WorkspaceID,
		ProjectID:    r.ProjectID,
		SubmissionID: r.SubmissionID,
		DecisionCode: r.DecisionCode,
		Comment:      r.Comment,
		ReviewedBy:   r.ReviewedBy,
		ReviewedAt:   r.ReviewedAt,
		CreatedAt:    r.CreatedAt,
		CreatedBy:    r.CreatedBy,
	}
}

func modelToReview(m reviewModel) deliverable.SubmissionReview {
	return deliverable.SubmissionReview{
		ID:           m.ID,
		WorkspaceID:  m.WorkspaceID,
		ProjectID:    m.ProjectID,
		SubmissionID: m.SubmissionID,
		DecisionCode: m.DecisionCode,
		Comment:      m.Comment,
		ReviewedBy:   m.ReviewedBy,
		ReviewedAt:   m.ReviewedAt,
		CreatedAt:    m.CreatedAt,
		CreatedBy:    m.CreatedBy,
	}
}
