package project

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrProjectNotFound = errors.New("project not found")
var ErrProjectRoleNotFound = errors.New("project role not found")
var ErrProjectPositionNotFound = errors.New("project position not found")
var ErrProjectPriorityNotFound = errors.New("project priority not found")
var ErrProjectCodeAlreadyTaken = errors.New("project code is already taken")
var ErrWorkspaceMembershipNotFound = errors.New("workspace membership not found")
var ErrProjectMemberAlreadyExists = errors.New("project member already exists")
var ErrProjectMemberNotFound = errors.New("project member not found")

type Repository interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context, repo Repository) error) error
	FindProjectRoleByCode(ctx context.Context, code ProjectRole) (*RoleMaster, error)
	FindProjectPositionByCode(ctx context.Context, code ProjectPosition) (*PositionMaster, error)
	FindProjectPriorityByCode(ctx context.Context, code ProjectPriority) (*PriorityMaster, error)
	ListProjectPositions(ctx context.Context) ([]PositionMaster, error)
	FindActiveWorkspaceMembershipByID(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, membershipID uuid.UUID) (*WorkspaceMemberCandidate, error)
	NextProjectCode(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, prefix string, year int, numberLength int) (string, error)
	CreateProject(ctx context.Context, project *Project) error
	CreateMember(ctx context.Context, member *Member) error
	UpdateProjectProfile(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, patch ProjectProfilePatch) error
	FindProjectByID(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID) (*ProjectWithMember, error)
	ListProjects(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, limit int, offset int) ([]ProjectWithMember, int, error)
	ListProjectMembers(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, limit int, offset int) ([]Member, int, error)
	FindProjectMemberByID(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, memberID uuid.UUID) (*Member, error)
	CountActiveProjectMembersByRole(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, role ProjectRole) (int, error)
	UpdateProjectMemberRole(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, memberID uuid.UUID, roleID uuid.UUID, updatedBy uuid.UUID, updatedAt time.Time) error
	RemoveProjectMember(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, memberID uuid.UUID, removedBy uuid.UUID, removedAt time.Time) error
	ListProjectMemberPositions(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, memberID uuid.UUID) ([]MemberPosition, error)
	ReplaceProjectMemberPositions(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, memberID uuid.UUID, positionIDs []uuid.UUID, createdBy uuid.UUID, createdAt time.Time) error
}
