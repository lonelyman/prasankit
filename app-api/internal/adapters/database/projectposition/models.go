// Package projectpositiondbrepo implements the projectposition domain ports against
// Postgres via GORM.
package projectpositiondbrepo

import (
	"time"

	"prasankit-api/internal/modules/projectposition"

	"github.com/google/uuid"
)

// projectPositionModel is the GORM model for project_positions.
type projectPositionModel struct {
	ID          uuid.UUID  `gorm:"column:id;primaryKey"`
	WorkspaceID uuid.UUID  `gorm:"column:workspace_id"`
	Code        string     `gorm:"column:code"`
	LabelTH     string     `gorm:"column:label_th"`
	LabelEN     string     `gorm:"column:label_en"`
	Description *string    `gorm:"column:description"`
	SortOrder   int        `gorm:"column:sort_order"`
	IsSystem    bool       `gorm:"column:is_system"`
	Status      string     `gorm:"column:status"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	CreatedBy   uuid.UUID  `gorm:"column:created_by"`
	UpdatedAt   time.Time  `gorm:"column:updated_at"`
	UpdatedBy   *uuid.UUID `gorm:"column:updated_by"`
}

func (projectPositionModel) TableName() string { return "project_positions" }

func toModel(p projectposition.ProjectPosition) projectPositionModel {
	return projectPositionModel{
		ID:          p.ID,
		WorkspaceID: p.WorkspaceID,
		Code:        p.Code,
		LabelTH:     p.LabelTH,
		LabelEN:     p.LabelEN,
		Description: p.Description,
		SortOrder:   p.SortOrder,
		IsSystem:    p.IsSystem,
		Status:      p.Status,
		CreatedAt:   p.CreatedAt,
		CreatedBy:   p.CreatedBy,
		UpdatedAt:   p.UpdatedAt,
		UpdatedBy:   p.UpdatedBy,
	}
}

func toDomain(m projectPositionModel) projectposition.ProjectPosition {
	return projectposition.ProjectPosition{
		ID:          m.ID,
		WorkspaceID: m.WorkspaceID,
		Code:        m.Code,
		LabelTH:     m.LabelTH,
		LabelEN:     m.LabelEN,
		Description: m.Description,
		SortOrder:   m.SortOrder,
		IsSystem:    m.IsSystem,
		Status:      m.Status,
		CreatedAt:   m.CreatedAt,
		CreatedBy:   m.CreatedBy,
		UpdatedAt:   m.UpdatedAt,
		UpdatedBy:   m.UpdatedBy,
	}
}
