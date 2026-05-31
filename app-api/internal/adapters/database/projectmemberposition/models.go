// Package projectmemberpositiondbrepo implements the projectmemberposition domain ports
// against Postgres via GORM.
package projectmemberpositiondbrepo

import (
	"time"

	"prasankit-api/internal/modules/projectmemberposition"

	"github.com/google/uuid"
)

// projectMemberPositionModel is the GORM model for project_member_positions.
// APPEND/DELETE-ONLY (D39): no updated_at/updated_by/status/removed_at.
type projectMemberPositionModel struct {
	ID                  uuid.UUID `gorm:"column:id;primaryKey"`
	WorkspaceID         uuid.UUID `gorm:"column:workspace_id"`
	ProjectMemberID     uuid.UUID `gorm:"column:project_member_id"`
	ProjectPositionCode string    `gorm:"column:project_position_code"`
	CreatedAt           time.Time `gorm:"column:created_at"`
	CreatedBy           uuid.UUID `gorm:"column:created_by"`
}

func (projectMemberPositionModel) TableName() string { return "project_member_positions" }

// positionWithLabelRow is the JOIN projection for ListByMember. FLAT struct (not embedded)
// because GORM's Find does not reliably scan an embedded model's column-tagged fields off an
// aliased SELECT (mirrors projectmember's memberWithDisplayNameRow).
type positionWithLabelRow struct {
	ID                  uuid.UUID `gorm:"column:id"`
	WorkspaceID         uuid.UUID `gorm:"column:workspace_id"`
	ProjectMemberID     uuid.UUID `gorm:"column:project_member_id"`
	ProjectPositionCode string    `gorm:"column:project_position_code"`
	CreatedAt           time.Time `gorm:"column:created_at"`
	CreatedBy           uuid.UUID `gorm:"column:created_by"`
	LabelTH             string    `gorm:"column:label_th"`
	LabelEN             string    `gorm:"column:label_en"`
	Status              string    `gorm:"column:status"`
}

func toModel(mp projectmemberposition.ProjectMemberPosition) projectMemberPositionModel {
	return projectMemberPositionModel{
		ID:                  mp.ID,
		WorkspaceID:         mp.WorkspaceID,
		ProjectMemberID:     mp.ProjectMemberID,
		ProjectPositionCode: mp.ProjectPositionCode,
		CreatedAt:           mp.CreatedAt,
		CreatedBy:           mp.CreatedBy,
	}
}

func rowToPositionWithLabel(r positionWithLabelRow) projectmemberposition.PositionWithLabel {
	return projectmemberposition.PositionWithLabel{
		ProjectMemberPosition: projectmemberposition.ProjectMemberPosition{
			ID:                  r.ID,
			WorkspaceID:         r.WorkspaceID,
			ProjectMemberID:     r.ProjectMemberID,
			ProjectPositionCode: r.ProjectPositionCode,
			CreatedAt:           r.CreatedAt,
			CreatedBy:           r.CreatedBy,
		},
		LabelTH: r.LabelTH,
		LabelEN: r.LabelEN,
		Status:  r.Status,
	}
}
