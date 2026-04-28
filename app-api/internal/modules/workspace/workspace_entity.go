package workspace

import (
	"time"

	"github.com/google/uuid"
)

type WorkspaceMode string

const (
	WorkspaceModeDemo       WorkspaceMode = "demo"
	WorkspaceModeProduction WorkspaceMode = "production"
)

type WorkspaceStatus string

const (
	WorkspaceStatusActive          WorkspaceStatus = "active"
	WorkspaceStatusSuspended       WorkspaceStatus = "suspended"
	WorkspaceStatusPendingDeletion WorkspaceStatus = "pending_deletion"
	WorkspaceStatusDeleted         WorkspaceStatus = "deleted"
)

type WorkspaceRole string

const (
	WorkspaceRoleOwner     WorkspaceRole = "owner"
	WorkspaceRoleAdmin     WorkspaceRole = "admin"
	WorkspaceRoleExecutive WorkspaceRole = "executive"
	WorkspaceRoleUser      WorkspaceRole = "user"
)

type MembershipStatus string

const (
	MembershipStatusActive    MembershipStatus = "active"
	MembershipStatusRemoved   MembershipStatus = "removed"
	MembershipStatusSuspended MembershipStatus = "suspended"
)

type WorkspaceRoleMaster struct {
	ID          uuid.UUID
	Code        WorkspaceRole
	Name        string
	Description string
	SortOrder   int
	IsSystem    bool
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Workspace struct {
	ID                    uuid.UUID
	TenantID              uuid.UUID
	Name                  string
	Slug                  string
	Mode                  WorkspaceMode
	Status                WorkspaceStatus
	ContactEmail          string
	OwnerUserAccountID    uuid.UUID
	EmailVerifiedRequired bool
	CreatedAt             time.Time
	CreatedBy             uuid.UUID
	UpdatedAt             time.Time
	UpdatedBy             *uuid.UUID
	PendingDeletionAt     *time.Time
	DeletedAt             *time.Time
	DeletedBy             *uuid.UUID
	HardDeletedAt         *time.Time
}

type Membership struct {
	ID            uuid.UUID
	TenantID      uuid.UUID
	WorkspaceID   uuid.UUID
	RoleID        uuid.UUID
	ProfileID     *uuid.UUID
	UserAccountID *uuid.UUID
	Role          WorkspaceRole
	Status        MembershipStatus
	StatusReason  string
	JoinedAt      *time.Time
	RemovedAt     *time.Time
	SuspendedAt   *time.Time
	CreatedAt     time.Time
	CreatedBy     *uuid.UUID
	UpdatedAt     time.Time
	UpdatedBy     *uuid.UUID
}

type WorkspaceWithMembership struct {
	Workspace  Workspace
	Membership Membership
}

type TenantContext struct {
	TenantID      uuid.UUID
	WorkspaceID   uuid.UUID
	WorkspaceSlug string
	MembershipID  uuid.UUID
	Role          WorkspaceRole
}
