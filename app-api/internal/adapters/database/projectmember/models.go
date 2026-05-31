// Package projectmemberdbrepo implements the projectmember domain ports against
// Postgres via GORM.
package projectmemberdbrepo

import (
	"time"

	"prasankit-api/internal/modules/projectmember"

	"github.com/google/uuid"
)

// projectMemberModel is the GORM model for project_members.
type projectMemberModel struct {
	ID                    uuid.UUID  `gorm:"column:id;primaryKey"`
	WorkspaceID           uuid.UUID  `gorm:"column:workspace_id"`
	ProjectID             uuid.UUID  `gorm:"column:project_id"`
	WorkspaceMembershipID uuid.UUID  `gorm:"column:workspace_membership_id"`
	ProjectRoleCode       string     `gorm:"column:project_role_code"`
	JoinedAt              time.Time  `gorm:"column:joined_at"`
	RemovedAt             *time.Time `gorm:"column:removed_at"`
	CreatedAt             time.Time  `gorm:"column:created_at"`
	CreatedBy             uuid.UUID  `gorm:"column:created_by"`
	UpdatedAt             time.Time  `gorm:"column:updated_at"`
	UpdatedBy             *uuid.UUID `gorm:"column:updated_by"`
}

func (projectMemberModel) TableName() string { return "project_members" }

// memberWithDisplayNameRow is the JOIN projection for ListByProject. It is a FLAT
// struct (not an embedded projectMemberModel): GORM's Find does not reliably scan an
// anonymous embedded model's column-tagged fields off an aliased SELECT (the embedded
// fields come back zero while top-level fields populate), so every column is declared
// here directly.
type memberWithDisplayNameRow struct {
	ID                    uuid.UUID  `gorm:"column:id"`
	WorkspaceID           uuid.UUID  `gorm:"column:workspace_id"`
	ProjectID             uuid.UUID  `gorm:"column:project_id"`
	WorkspaceMembershipID uuid.UUID  `gorm:"column:workspace_membership_id"`
	ProjectRoleCode       string     `gorm:"column:project_role_code"`
	JoinedAt              time.Time  `gorm:"column:joined_at"`
	RemovedAt             *time.Time `gorm:"column:removed_at"`
	CreatedAt             time.Time  `gorm:"column:created_at"`
	CreatedBy             uuid.UUID  `gorm:"column:created_by"`
	UpdatedAt             time.Time  `gorm:"column:updated_at"`
	UpdatedBy             *uuid.UUID `gorm:"column:updated_by"`
	DisplayName           string     `gorm:"column:display_name"`
}

func projectMemberToModel(m projectmember.ProjectMember) projectMemberModel {
	return projectMemberModel{
		ID:                    m.ID,
		WorkspaceID:           m.WorkspaceID,
		ProjectID:             m.ProjectID,
		WorkspaceMembershipID: m.WorkspaceMembershipID,
		ProjectRoleCode:       m.ProjectRoleCode,
		JoinedAt:              m.JoinedAt,
		RemovedAt:             m.RemovedAt,
		CreatedAt:             m.CreatedAt,
		CreatedBy:             m.CreatedBy,
		UpdatedAt:             m.UpdatedAt,
		UpdatedBy:             m.UpdatedBy,
	}
}

func modelToProjectMember(m projectMemberModel) projectmember.ProjectMember {
	return projectmember.ProjectMember{
		ID:                    m.ID,
		WorkspaceID:           m.WorkspaceID,
		ProjectID:             m.ProjectID,
		WorkspaceMembershipID: m.WorkspaceMembershipID,
		ProjectRoleCode:       m.ProjectRoleCode,
		JoinedAt:              m.JoinedAt,
		RemovedAt:             m.RemovedAt,
		CreatedAt:             m.CreatedAt,
		CreatedBy:             m.CreatedBy,
		UpdatedAt:             m.UpdatedAt,
		UpdatedBy:             m.UpdatedBy,
	}
}

func rowToMemberWithDisplayName(r memberWithDisplayNameRow) projectmember.MemberWithDisplayName {
	return projectmember.MemberWithDisplayName{
		ProjectMember: projectmember.ProjectMember{
			ID:                    r.ID,
			WorkspaceID:           r.WorkspaceID,
			ProjectID:             r.ProjectID,
			WorkspaceMembershipID: r.WorkspaceMembershipID,
			ProjectRoleCode:       r.ProjectRoleCode,
			JoinedAt:              r.JoinedAt,
			RemovedAt:             r.RemovedAt,
			CreatedAt:             r.CreatedAt,
			CreatedBy:             r.CreatedBy,
			UpdatedAt:             r.UpdatedAt,
			UpdatedBy:             r.UpdatedBy,
		},
		DisplayName: r.DisplayName,
	}
}
