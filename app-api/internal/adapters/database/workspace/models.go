// Package workspacedbrepo implements the workspace domain ports against Postgres via GORM.
package workspacedbrepo

import (
	"time"

	"github.com/google/uuid"
)

// workspaceModel is the GORM model for workspaces.
type workspaceModel struct {
	ID                  uuid.UUID  `gorm:"type:uuid;primaryKey"`
	WorkspaceName       string     `gorm:"column:workspace_name;not null"`
	Slug                string     `gorm:"column:slug;not null"`
	WorkspaceStatusCode string     `gorm:"column:workspace_status_code;not null"`
	ContactEmail        string     `gorm:"column:contact_email;not null"`
	OwnerUserAccountID  uuid.UUID  `gorm:"type:uuid;column:owner_user_account_id;not null"`
	CreatedAt           time.Time  `gorm:"column:created_at;not null"`
	CreatedBy           *uuid.UUID `gorm:"type:uuid;column:created_by"`
	UpdatedAt           time.Time  `gorm:"column:updated_at;not null"`
	UpdatedBy           *uuid.UUID `gorm:"type:uuid;column:updated_by"`
	PendingDeletionAt   *time.Time `gorm:"column:pending_deletion_at"`
	DeletedAt           *time.Time `gorm:"column:deleted_at"`
	DeletedBy           *uuid.UUID `gorm:"type:uuid;column:deleted_by"`
}

func (workspaceModel) TableName() string { return "workspaces" }

// membershipModel is the GORM model for workspace_memberships.
type membershipModel struct {
	ID                     uuid.UUID  `gorm:"type:uuid;primaryKey"`
	WorkspaceID            uuid.UUID  `gorm:"type:uuid;column:workspace_id;not null"`
	UserAccountID          uuid.UUID  `gorm:"type:uuid;column:user_account_id;not null"`
	OrgRoleCode            string     `gorm:"column:org_role_code;not null"`
	MembershipStatusCode   string     `gorm:"column:membership_status_code;not null"`
	InvitedByUserAccountID *uuid.UUID `gorm:"type:uuid;column:invited_by_user_account_id"`
	JoinedAt               *time.Time `gorm:"column:joined_at"`
	CreatedAt              time.Time  `gorm:"column:created_at;not null"`
	CreatedBy              *uuid.UUID `gorm:"type:uuid;column:created_by"`
	UpdatedAt              time.Time  `gorm:"column:updated_at;not null"`
	UpdatedBy              *uuid.UUID `gorm:"type:uuid;column:updated_by"`
}

func (membershipModel) TableName() string { return "workspace_memberships" }

// workspaceWithRoleRow is used for the JOIN query in ListActiveWorkspacesByAccount.
type workspaceWithRoleRow struct {
	workspaceModel
	OrgRoleCode  string    `gorm:"column:org_role_code"`
	MembershipID uuid.UUID `gorm:"column:membership_id"`
}

// membershipWithDisplayNameRow is the flat JOIN projection for ListActiveWithDisplayName
// (workspace_memberships → user_accounts). Maps to workspace.MembershipWithDisplayName.
type membershipWithDisplayNameRow struct {
	ID          uuid.UUID `gorm:"column:id"`
	DisplayName string    `gorm:"column:display_name"`
	OrgRoleCode string    `gorm:"column:org_role_code"`
}
