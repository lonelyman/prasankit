package project

import (
	"time"

	"github.com/google/uuid"
)

type ProjectType string

const (
	ProjectTypeInternal ProjectType = "internal"
	ProjectTypeClient   ProjectType = "client"
)

type ProjectStatus string

const (
	ProjectStatusDraft       ProjectStatus = "draft"
	ProjectStatusPlanning    ProjectStatus = "planning"
	ProjectStatusProposal    ProjectStatus = "proposal"
	ProjectStatusActive      ProjectStatus = "active"
	ProjectStatusClosing     ProjectStatus = "closing"
	ProjectStatusMaintenance ProjectStatus = "maintenance"
	ProjectStatusClosed      ProjectStatus = "closed"
	ProjectStatusArchived    ProjectStatus = "archived"
)

type ProjectRole string

const (
	ProjectRoleOwner   ProjectRole = "project_owner"
	ProjectRoleManager ProjectRole = "project_manager"
	ProjectRoleMember  ProjectRole = "member"
	ProjectRoleFinance ProjectRole = "finance"
	ProjectRoleViewer  ProjectRole = "viewer"
)

type ProjectMemberStatus string

const (
	ProjectMemberStatusActive  ProjectMemberStatus = "active"
	ProjectMemberStatusRemoved ProjectMemberStatus = "removed"
)

type ProjectPriority string

const (
	ProjectPriorityLow    ProjectPriority = "low"
	ProjectPriorityMedium ProjectPriority = "medium"
	ProjectPriorityHigh   ProjectPriority = "high"
)

type ProjectPosition string

const (
	ProjectPositionLead            ProjectPosition = "project_lead"
	ProjectPositionBusinessAnalyst ProjectPosition = "business_analyst"
	ProjectPositionDeveloper       ProjectPosition = "developer"
	ProjectPositionDesigner        ProjectPosition = "designer"
	ProjectPositionTester          ProjectPosition = "tester"
	ProjectPositionDevOps          ProjectPosition = "devops"
	ProjectPositionFinanceContact  ProjectPosition = "finance_contact"
	ProjectPositionStakeholder     ProjectPosition = "stakeholder"
)

type RoleMaster struct {
	ID          uuid.UUID
	Code        ProjectRole
	Name        string
	Description string
	SortOrder   int
	IsSystem    bool
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PriorityMaster struct {
	ID        uuid.UUID
	Code      ProjectPriority
	Name      string
	SortOrder int
	IsSystem  bool
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type PositionMaster struct {
	ID          uuid.UUID
	Code        ProjectPosition
	Name        string
	Description string
	SortOrder   int
	IsSystem    bool
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Project struct {
	ID                     uuid.UUID
	TenantID               uuid.UUID
	WorkspaceID            uuid.UUID
	Code                   string
	Name                   string
	Type                   ProjectType
	Status                 ProjectStatus
	PriorityID             uuid.UUID
	Priority               ProjectPriority
	Description            string
	ClientOrRequestingUnit string
	ScopeOrObjective       string
	CreatedBy              uuid.UUID
	CreatedAt              time.Time
	UpdatedBy              *uuid.UUID
	UpdatedAt              time.Time
	DeletedBy              *uuid.UUID
	DeletedAt              *time.Time
	ArchivedAt             *time.Time
}

type ProjectProfilePatch struct {
	Name                   *string
	Type                   *ProjectType
	PriorityID             *uuid.UUID
	Description            *string
	ClientOrRequestingUnit *string
	ScopeOrObjective       *string
	UpdatedBy              uuid.UUID
	UpdatedAt              time.Time
}

type WorkspaceMemberCandidate struct {
	MembershipID  uuid.UUID
	TenantID      uuid.UUID
	WorkspaceID   uuid.UUID
	ProfileID     *uuid.UUID
	UserAccountID *uuid.UUID
}

type Member struct {
	ID                    uuid.UUID
	TenantID              uuid.UUID
	WorkspaceID           uuid.UUID
	ProjectID             uuid.UUID
	WorkspaceMembershipID uuid.UUID
	ProfileID             *uuid.UUID
	UserAccountID         *uuid.UUID
	RoleID                uuid.UUID
	Role                  ProjectRole
	Status                ProjectMemberStatus
	JoinedAt              *time.Time
	RemovedAt             *time.Time
	CreatedBy             *uuid.UUID
	CreatedAt             time.Time
	UpdatedBy             *uuid.UUID
	UpdatedAt             time.Time
}

type MemberPosition struct {
	ID              uuid.UUID
	TenantID        uuid.UUID
	WorkspaceID     uuid.UUID
	ProjectID       uuid.UUID
	ProjectMemberID uuid.UUID
	PositionID      uuid.UUID
	Position        ProjectPosition
	Name            string
	CreatedBy       *uuid.UUID
	CreatedAt       time.Time
}

type ProjectWithMember struct {
	Project Project
	Member  Member
}
