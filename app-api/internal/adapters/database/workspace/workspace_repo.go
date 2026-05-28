package workspacedbrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	auditdbrepo "prasankit-api/internal/adapters/database/audit"
	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/workspace"

	"gorm.io/gorm"
)

// WorkspaceRepo implements workspace.WorkspaceRepository against Postgres via GORM.
type WorkspaceRepo struct {
	db        *gorm.DB
	auditRepo *auditdbrepo.AuditRepo
}

// NewWorkspaceRepo constructs a WorkspaceRepo holding only the base *gorm.DB.
func NewWorkspaceRepo(db *gorm.DB, auditRepo *auditdbrepo.AuditRepo) *WorkspaceRepo {
	return &WorkspaceRepo{db: db, auditRepo: auditRepo}
}

// CreateWorkspaceWithOwner atomically inserts (1) workspace, (2) owner membership,
// (3) audit_log entry in a single transaction. Any failure rolls back all three.
// The audit_logs insert is delegated to auditRepo.LogTx to keep that knowledge
// inside the audit adapter.
func (r *WorkspaceRepo) CreateWorkspaceWithOwner(
	ctx context.Context,
	ws workspace.Workspace,
	m workspace.Membership,
	entry audit.Entry,
) error {
	wsModel := workspaceToModel(ws)
	memberModel := membershipToModel(m)

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&wsModel).Error; err != nil {
			return err
		}
		if err := tx.Create(&memberModel).Error; err != nil {
			return err
		}
		if err := r.auditRepo.LogTx(tx, entry); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if isUniqueViolation(err) {
			return workspace.ErrSlugTaken
		}
		return fmt.Errorf("create workspace with owner: %w", err)
	}
	return nil
}

// FindBySlugActive returns an active (non-deleted) workspace by slug.
// Returns nil, nil when not found.
func (r *WorkspaceRepo) FindBySlugActive(ctx context.Context, slug string) (*workspace.Workspace, error) {
	var m workspaceModel
	err := r.db.WithContext(ctx).
		Where("slug = ? AND deleted_at IS NULL", slug).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find workspace by slug: %w", err)
	}
	ws := modelToWorkspace(m)
	return &ws, nil
}

// isUniqueViolation detects Postgres unique constraint violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}
