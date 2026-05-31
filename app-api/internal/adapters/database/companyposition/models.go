// Package companypositiondbrepo implements the companyposition domain ports against
// Postgres via GORM.
package companypositiondbrepo

import (
	"time"

	"prasankit-api/internal/modules/companyposition"

	"github.com/google/uuid"
)

// companyPositionModel is the GORM model for company_positions.
type companyPositionModel struct {
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

func (companyPositionModel) TableName() string { return "company_positions" }

func toModel(p companyposition.CompanyPosition) companyPositionModel {
	return companyPositionModel{
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

func toDomain(m companyPositionModel) companyposition.CompanyPosition {
	return companyposition.CompanyPosition{
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
