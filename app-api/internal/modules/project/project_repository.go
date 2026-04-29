package project

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrProjectNotFound = errors.New("project not found")
var ErrProjectRoleNotFound = errors.New("project role not found")
var ErrProjectPriorityNotFound = errors.New("project priority not found")
var ErrProjectCodeAlreadyTaken = errors.New("project code is already taken")

type Repository interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context, repo Repository) error) error
	FindProjectRoleByCode(ctx context.Context, code ProjectRole) (*RoleMaster, error)
	FindProjectPriorityByCode(ctx context.Context, code ProjectPriority) (*PriorityMaster, error)
	NextProjectCode(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, prefix string, year int, numberLength int) (string, error)
	CreateProject(ctx context.Context, project *Project) error
	CreateMember(ctx context.Context, member *Member) error
	UpdateProjectProfile(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, patch ProjectProfilePatch) error
	FindProjectByID(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID) (*ProjectWithMember, error)
	ListProjects(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, limit int, offset int) ([]ProjectWithMember, int, error)
	ListProjectMembers(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, limit int, offset int) ([]Member, int, error)
}
