package workspace

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrWorkspaceNotFound = errors.New("workspace not found")
var ErrMembershipNotFound = errors.New("workspace membership not found")
var ErrWorkspaceRoleNotFound = errors.New("workspace role not found")
var ErrWorkspaceSlugAlreadyTaken = errors.New("workspace slug is already taken")

type Repository interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context, repo Repository) error) error
	FindWorkspaceBySlug(ctx context.Context, slug string) (*Workspace, error)
	FindWorkspaceRoleByCode(ctx context.Context, code WorkspaceRole) (*WorkspaceRoleMaster, error)
	CreateWorkspace(ctx context.Context, workspace *Workspace) error
	CreateMembership(ctx context.Context, membership *Membership) error
	FindActiveMembership(ctx context.Context, tenantID uuid.UUID, userAccountID uuid.UUID) (*Membership, error)
	FindActiveWorkspaceMembershipBySlug(ctx context.Context, slug string, userAccountID uuid.UUID) (*WorkspaceWithMembership, error)
	ListWorkspacesByUserAccountID(ctx context.Context, userAccountID uuid.UUID, limit int, offset int) ([]WorkspaceWithMembership, int, error)
}
