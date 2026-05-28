package workspacedbrepo

import (
	"context"
	"errors"
	"fmt"

	"prasankit-api/internal/modules/workspace"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MembershipRepo implements workspace.MembershipRepository against Postgres via GORM.
type MembershipRepo struct {
	db *gorm.DB
}

// NewMembershipRepo constructs a MembershipRepo holding only the base *gorm.DB.
func NewMembershipRepo(db *gorm.DB) *MembershipRepo {
	return &MembershipRepo{db: db}
}

// FindActiveByWorkspaceAndAccount returns the active membership for the given
// workspace + account pair. Isolation invariant: workspace_id is always in WHERE.
// Returns nil, nil when not found.
func (r *MembershipRepo) FindActiveByWorkspaceAndAccount(
	ctx context.Context,
	workspaceID, accountID uuid.UUID,
) (*workspace.Membership, error) {
	var m membershipModel
	err := r.db.WithContext(ctx).
		Where(
			"workspace_id = ? AND user_account_id = ? AND membership_status_code = ?",
			workspaceID, accountID, workspace.MembershipStatusActive,
		).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find active membership: %w", err)
	}
	ms := modelToMembership(m)
	return &ms, nil
}

// ListActiveWorkspacesByAccount returns all workspaces (with the account's role) where
// the account has an active membership and the workspace is active and not deleted.
// Scoped by accountID — NOT by workspace_id (correct: returns "my workspaces").
func (r *MembershipRepo) ListActiveWorkspacesByAccount(
	ctx context.Context,
	accountID uuid.UUID,
) ([]workspace.WorkspaceWithRole, error) {
	// Use raw SQL for the join to avoid GORM ambiguity on column names.
	const query = `
		SELECT
			w.id,
			w.workspace_name,
			w.slug,
			w.workspace_status_code,
			w.contact_email,
			w.owner_user_account_id,
			w.created_at,
			wm.org_role_code,
			wm.id AS membership_id
		FROM workspaces w
		INNER JOIN workspace_memberships wm
			ON wm.workspace_id = w.id
			AND wm.user_account_id = ?
			AND wm.membership_status_code = 'active'
		WHERE
			w.workspace_status_code = 'active'
			AND w.deleted_at IS NULL
		ORDER BY w.created_at ASC
	`

	type resultRow struct {
		ID                  uuid.UUID `gorm:"column:id"`
		WorkspaceName       string    `gorm:"column:workspace_name"`
		Slug                string    `gorm:"column:slug"`
		WorkspaceStatusCode string    `gorm:"column:workspace_status_code"`
		ContactEmail        string    `gorm:"column:contact_email"`
		OwnerUserAccountID  uuid.UUID `gorm:"column:owner_user_account_id"`
		OrgRoleCode         string    `gorm:"column:org_role_code"`
		MembershipID        uuid.UUID `gorm:"column:membership_id"`
	}

	var rows []resultRow
	if err := r.db.WithContext(ctx).Raw(query, accountID).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("list workspaces by account: %w", err)
	}

	result := make([]workspace.WorkspaceWithRole, len(rows))
	for i, row := range rows {
		result[i] = workspace.WorkspaceWithRole{
			Workspace: workspace.Workspace{
				ID:                  row.ID,
				WorkspaceName:       row.WorkspaceName,
				Slug:                row.Slug,
				WorkspaceStatusCode: row.WorkspaceStatusCode,
				ContactEmail:        row.ContactEmail,
				OwnerUserAccountID:  row.OwnerUserAccountID,
			},
			OrgRoleCode: row.OrgRoleCode,
		}
	}
	return result, nil
}

// ListByWorkspace returns all memberships for the given workspace_id.
// Isolation invariant: workspace_id is always in WHERE.
func (r *MembershipRepo) ListByWorkspace(
	ctx context.Context,
	workspaceID uuid.UUID,
) ([]workspace.Membership, error) {
	var models []membershipModel
	err := r.db.WithContext(ctx).
		Where("workspace_id = ?", workspaceID).
		Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("list memberships by workspace: %w", err)
	}
	result := make([]workspace.Membership, len(models))
	for i, m := range models {
		result[i] = modelToMembership(m)
	}
	return result, nil
}
