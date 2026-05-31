package projectmemberdbrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	auditdbrepo "prasankit-api/internal/adapters/database/audit"
	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/projectmember"
	"prasankit-api/internal/modules/workspace"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProjectMemberRepo implements projectmember.ProjectMemberRepository against Postgres via GORM.
type ProjectMemberRepo struct {
	db        *gorm.DB
	auditRepo *auditdbrepo.AuditRepo
}

// NewProjectMemberRepo constructs a ProjectMemberRepo holding only the base *gorm.DB.
func NewProjectMemberRepo(db *gorm.DB, auditRepo *auditdbrepo.AuditRepo) *ProjectMemberRepo {
	return &ProjectMemberRepo{db: db, auditRepo: auditRepo}
}

// AddWithAudit atomically inserts a project_member + audit_logs entry in one transaction.
// uq_project_members_active violation -> ErrAlreadyMember. Composite FK violation
// (cross-ws membership / project) propagates as the raw wrapped error — the DB backstop.
func (r *ProjectMemberRepo) AddWithAudit(ctx context.Context, m projectmember.ProjectMember, entry audit.Entry) error {
	model := projectMemberToModel(m)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&model).Error; err != nil {
			if isUniqueViolation(err) {
				return projectmember.ErrAlreadyMember
			}
			return err
		}
		return r.auditRepo.LogTx(tx, entry)
	})
	if err != nil {
		if errors.Is(err, projectmember.ErrAlreadyMember) {
			return err
		}
		return fmt.Errorf("add project_member: %w", err)
	}
	return nil
}

// RemoveWithAudit atomically sets removed_at + audit_logs entry in one transaction.
// The Updates map contains ONLY {removed_at, updated_at, updated_by} (never FK columns).
func (r *ProjectMemberRepo) RemoveWithAudit(ctx context.Context, workspaceID, projectID, memberID, removedBy uuid.UUID, entry audit.Entry) error {
	now := time.Now().UTC()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&projectMemberModel{}).
			Where("workspace_id = ? AND project_id = ? AND id = ? AND removed_at IS NULL", workspaceID, projectID, memberID).
			Updates(map[string]any{
				"removed_at": now,
				"updated_at": now,
				"updated_by": removedBy,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return projectmember.ErrProjectMemberNotFound
		}
		return r.auditRepo.LogTx(tx, entry)
	})
	if err != nil {
		if errors.Is(err, projectmember.ErrProjectMemberNotFound) {
			return err
		}
		return fmt.Errorf("remove project_member: %w", err)
	}
	return nil
}

// ChangeRoleWithAudit atomically updates project_role_code + audit_logs entry in one transaction.
// The Updates map contains ONLY {project_role_code, updated_at, updated_by} (never FK columns),
// so the 23503->ErrInvalidRoleCode mapping is safe: only the project_role_code single-column FK
// can fire here — the composite (cross-ws) FKs are untouched.
func (r *ProjectMemberRepo) ChangeRoleWithAudit(ctx context.Context, workspaceID, projectID, memberID uuid.UUID, newRoleCode string, updatedBy uuid.UUID, entry audit.Entry) error {
	now := time.Now().UTC()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&projectMemberModel{}).
			Where("workspace_id = ? AND project_id = ? AND id = ? AND removed_at IS NULL", workspaceID, projectID, memberID).
			Updates(map[string]any{
				"project_role_code": newRoleCode,
				"updated_at":        now,
				"updated_by":        updatedBy,
			})
		if result.Error != nil {
			if isForeignKeyViolation(result.Error) {
				return projectmember.ErrInvalidRoleCode
			}
			return result.Error
		}
		if result.RowsAffected == 0 {
			return projectmember.ErrProjectMemberNotFound
		}
		return r.auditRepo.LogTx(tx, entry)
	})
	if err != nil {
		if errors.Is(err, projectmember.ErrInvalidRoleCode) || errors.Is(err, projectmember.ErrProjectMemberNotFound) {
			return err
		}
		return fmt.Errorf("change project_member role: %w", err)
	}
	return nil
}

// ListByProject returns active members of the project, scoped to the project's workspace,
// JOINed to workspace_memberships + user_accounts for display_name. The ws-suspended-member
// filter (§M2.3.4 line 693) is applied via wm.membership_status_code = 'active'. The
// wm.workspace_id = pm.workspace_id JOIN predicate is load-bearing: it prevents a multi-ws
// user's display_name being pulled via a sibling-workspace membership.
func (r *ProjectMemberRepo) ListByProject(ctx context.Context, workspaceID, projectID uuid.UUID) ([]projectmember.MemberWithDisplayName, error) {
	var rows []memberWithDisplayNameRow
	err := r.db.WithContext(ctx).
		Table("project_members AS pm").
		Select(`pm.id, pm.workspace_id, pm.project_id, pm.workspace_membership_id,
			pm.project_role_code, pm.joined_at, pm.removed_at, pm.created_at,
			pm.created_by, pm.updated_at, pm.updated_by, ua.display_name AS display_name`).
		Joins("INNER JOIN workspace_memberships wm ON wm.workspace_id = pm.workspace_id AND wm.id = pm.workspace_membership_id").
		Joins("INNER JOIN user_accounts ua ON ua.id = wm.user_account_id").
		Where("pm.workspace_id = ? AND pm.project_id = ? AND pm.removed_at IS NULL AND wm.membership_status_code = ?",
			workspaceID, projectID, workspace.MembershipStatusActive).
		Order("pm.joined_at ASC").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list project_members: %w", err)
	}
	out := make([]projectmember.MemberWithDisplayName, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToMemberWithDisplayName(row))
	}
	return out, nil
}

// FindActiveByID returns the active project_member (removed_at IS NULL) scoped to workspace_id.
// gorm.ErrRecordNotFound → (nil, nil). Other errors propagate.
func (r *ProjectMemberRepo) FindActiveByID(ctx context.Context, workspaceID, projectID, memberID uuid.UUID) (*projectmember.ProjectMember, error) {
	var m projectMemberModel
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND project_id = ? AND id = ? AND removed_at IS NULL", workspaceID, projectID, memberID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find project_member: %w", err)
	}
	pm := modelToProjectMember(m)
	return &pm, nil
}

// FindByID returns the project_member INCLUDING removed rows (RemovedAt populated when set),
// scoped to workspace_id. This is FindActiveByID minus the removed_at IS NULL predicate (6b-2).
// gorm.ErrRecordNotFound → (nil, nil). Other errors propagate.
func (r *ProjectMemberRepo) FindByID(ctx context.Context, workspaceID, projectID, memberID uuid.UUID) (*projectmember.ProjectMember, error) {
	var m projectMemberModel
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND project_id = ? AND id = ?", workspaceID, projectID, memberID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find project_member: %w", err)
	}
	pm := modelToProjectMember(m)
	return &pm, nil
}

// IsActiveWorkspaceMember returns true when the membership is an ACTIVE membership of this workspace.
func (r *ProjectMemberRepo) IsActiveWorkspaceMember(ctx context.Context, workspaceID, membershipID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("workspace_memberships").
		Where("workspace_id = ? AND id = ? AND membership_status_code = ?", workspaceID, membershipID, workspace.MembershipStatusActive).
		Limit(1).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("check active workspace member: %w", err)
	}
	return count > 0, nil
}

// ProjectExistsForWorkspace returns true when a non-deleted project exists in this workspace.
func (r *ProjectMemberRepo) ProjectExistsForWorkspace(ctx context.Context, workspaceID, projectID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("projects").
		Where("workspace_id = ? AND id = ? AND deleted_at IS NULL", workspaceID, projectID).
		Limit(1).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("check project exists: %w", err)
	}
	return count > 0, nil
}

// isUniqueViolation detects Postgres unique constraint violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}

// isForeignKeyViolation detects Postgres foreign-key violation (SQLSTATE 23503).
func isForeignKeyViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23503")
}
