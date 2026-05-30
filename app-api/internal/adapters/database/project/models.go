// Package projectdbrepo implements the project domain ports against Postgres via GORM.
package projectdbrepo

import (
	"time"

	"prasankit-api/internal/modules/project"

	"github.com/google/uuid"
)

// projectModel is the GORM model for projects.
type projectModel struct {
	ID                   uuid.UUID  `gorm:"column:id;primaryKey"`
	WorkspaceID          uuid.UUID  `gorm:"column:workspace_id"`
	ProjectName          string     `gorm:"column:project_name"`
	Slug                 *string    `gorm:"column:slug"`
	ProjectTypeCode      string     `gorm:"column:project_type_code"`
	ProjectStatusCode    string     `gorm:"column:project_status_code"`
	OwnerProjectMemberID *uuid.UUID `gorm:"column:owner_project_member_id"`
	RequestingUnit       *string    `gorm:"column:requesting_unit"`
	Description          *string    `gorm:"column:description"`
	StartDate            *time.Time `gorm:"column:start_date"`
	EndDate              *time.Time `gorm:"column:end_date"`
	CreatedAt            time.Time  `gorm:"column:created_at"`
	CreatedBy            uuid.UUID  `gorm:"column:created_by"`
	UpdatedAt            time.Time  `gorm:"column:updated_at"`
	UpdatedBy            *uuid.UUID `gorm:"column:updated_by"`
	DeletedAt            *time.Time `gorm:"column:deleted_at"`
	DeletedBy            *uuid.UUID `gorm:"column:deleted_by"`
}

func (projectModel) TableName() string { return "projects" }

func projectToModel(p project.Project) projectModel {
	return projectModel{
		ID:                   p.ID,
		WorkspaceID:          p.WorkspaceID,
		ProjectName:          p.ProjectName,
		Slug:                 p.Slug,
		ProjectTypeCode:      p.ProjectTypeCode,
		ProjectStatusCode:    p.ProjectStatusCode,
		OwnerProjectMemberID: p.OwnerProjectMemberID,
		RequestingUnit:       p.RequestingUnit,
		Description:          p.Description,
		StartDate:            p.StartDate,
		EndDate:              p.EndDate,
		CreatedAt:            p.CreatedAt,
		CreatedBy:            p.CreatedBy,
		UpdatedAt:            p.UpdatedAt,
		UpdatedBy:            p.UpdatedBy,
		DeletedAt:            p.DeletedAt,
		DeletedBy:            p.DeletedBy,
	}
}

func modelToProject(m projectModel) project.Project {
	return project.Project{
		ID:                   m.ID,
		WorkspaceID:          m.WorkspaceID,
		ProjectName:          m.ProjectName,
		Slug:                 m.Slug,
		ProjectTypeCode:      m.ProjectTypeCode,
		ProjectStatusCode:    m.ProjectStatusCode,
		OwnerProjectMemberID: m.OwnerProjectMemberID,
		RequestingUnit:       m.RequestingUnit,
		Description:          m.Description,
		StartDate:            m.StartDate,
		EndDate:              m.EndDate,
		CreatedAt:            m.CreatedAt,
		CreatedBy:            m.CreatedBy,
		UpdatedAt:            m.UpdatedAt,
		UpdatedBy:            m.UpdatedBy,
		DeletedAt:            m.DeletedAt,
		DeletedBy:            m.DeletedBy,
	}
}
