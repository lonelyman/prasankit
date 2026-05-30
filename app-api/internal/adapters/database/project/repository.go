package projectdbrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	auditdbrepo "prasankit-api/internal/adapters/database/audit"
	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/project"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProjectRepo implements project.ProjectRepository against Postgres via GORM.
type ProjectRepo struct {
	db        *gorm.DB
	auditRepo *auditdbrepo.AuditRepo
}

// NewProjectRepo constructs a ProjectRepo holding only the base *gorm.DB.
func NewProjectRepo(db *gorm.DB, auditRepo *auditdbrepo.AuditRepo) *ProjectRepo {
	return &ProjectRepo{db: db, auditRepo: auditRepo}
}

// CreateWithAudit atomically inserts a project + audit_logs entry in one transaction.
// Returns project.ErrSlugTaken on uq_projects_workspace_slug_active violation.
// FK violations (project_status_code / project_type_code) propagate as the raw wrapped
// error — the service pre-check covers the normal case; the DB error is the backstop.
func (r *ProjectRepo) CreateWithAudit(ctx context.Context, p project.Project, entry audit.Entry) error {
	model := projectToModel(p)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&model).Error; err != nil {
			if isUniqueViolation(err) {
				return project.ErrSlugTaken
			}
			return err
		}
		return r.auditRepo.LogTx(tx, entry)
	})
	if err != nil {
		if errors.Is(err, project.ErrSlugTaken) {
			return err
		}
		return fmt.Errorf("create project: %w", err)
	}
	return nil
}

// FindByIDForWorkspace returns the project with WHERE workspace_id = ? AND id = ? AND deleted_at IS NULL.
// gorm.ErrRecordNotFound → (nil, nil). Other errors propagate.
func (r *ProjectRepo) FindByIDForWorkspace(ctx context.Context, workspaceID, projectID uuid.UUID) (*project.Project, error) {
	var m projectModel
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND id = ? AND deleted_at IS NULL", workspaceID, projectID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find project: %w", err)
	}
	p := modelToProject(m)
	return &p, nil
}

// ListByWorkspace returns the page of projects for the given workspace.
func (r *ProjectRepo) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, opts project.ListOptions) ([]project.Project, int64, error) {
	tx := r.db.WithContext(ctx).Model(&projectModel{}).
		Where("workspace_id = ? AND deleted_at IS NULL", workspaceID)
	if opts.StatusCode != "" {
		tx = tx.Where("project_status_code = ?", opts.StatusCode)
	}
	if opts.TypeCode != "" {
		tx = tx.Where("project_type_code = ?", opts.TypeCode)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count projects: %w", err)
	}

	if total == 0 {
		return []project.Project{}, 0, nil
	}

	var rows []projectModel
	offset := (opts.Page - 1) * opts.Limit
	if err := tx.Order("created_at DESC").Limit(opts.Limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list projects: %w", err)
	}

	out := make([]project.Project, 0, len(rows))
	for _, m := range rows {
		out = append(out, modelToProject(m))
	}
	return out, total, nil
}

// UpdateWithAudit atomically updates a project + audit_logs entry in one transaction.
// project_status_code is intentionally NOT in the update map (status changes use ChangeStatusWithAudit).
func (r *ProjectRepo) UpdateWithAudit(ctx context.Context, p project.Project, entry audit.Entry) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&projectModel{}).
			Where("workspace_id = ? AND id = ? AND deleted_at IS NULL", p.WorkspaceID, p.ID).
			Updates(map[string]any{
				"project_name":      p.ProjectName,
				"slug":              p.Slug,
				"project_type_code": p.ProjectTypeCode,
				"requesting_unit":   p.RequestingUnit,
				"description":       p.Description,
				"start_date":        p.StartDate,
				"end_date":          p.EndDate,
				"updated_at":        p.UpdatedAt,
				"updated_by":        p.UpdatedBy,
			})
		if result.Error != nil {
			if isUniqueViolation(result.Error) {
				return project.ErrSlugTaken
			}
			return result.Error
		}
		if result.RowsAffected == 0 {
			return project.ErrProjectNotFound
		}
		return r.auditRepo.LogTx(tx, entry)
	})
	if err != nil {
		if errors.Is(err, project.ErrSlugTaken) || errors.Is(err, project.ErrProjectNotFound) {
			return err
		}
		return fmt.Errorf("update project: %w", err)
	}
	return nil
}

// SoftDeleteWithAudit atomically soft-deletes a project + audit_logs entry.
func (r *ProjectRepo) SoftDeleteWithAudit(ctx context.Context, workspaceID, projectID, deletedBy uuid.UUID, entry audit.Entry) error {
	now := time.Now().UTC()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&projectModel{}).
			Where("workspace_id = ? AND id = ? AND deleted_at IS NULL", workspaceID, projectID).
			Updates(map[string]any{
				"deleted_at": now,
				"deleted_by": deletedBy,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return project.ErrProjectNotFound
		}
		return r.auditRepo.LogTx(tx, entry)
	})
	if err != nil {
		if errors.Is(err, project.ErrProjectNotFound) {
			return err
		}
		return fmt.Errorf("soft delete project: %w", err)
	}
	return nil
}

// ChangeStatusWithAudit atomically updates project_status_code + audit_logs entry.
// SQLSTATE 23503 (FK violation) on project_status_code → project.ErrInvalidStatusCode.
func (r *ProjectRepo) ChangeStatusWithAudit(ctx context.Context, workspaceID, projectID uuid.UUID, newStatusCode string, updatedBy uuid.UUID, entry audit.Entry) error {
	now := time.Now().UTC()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&projectModel{}).
			Where("workspace_id = ? AND id = ? AND deleted_at IS NULL", workspaceID, projectID).
			Updates(map[string]any{
				"project_status_code": newStatusCode,
				"updated_at":          now,
				"updated_by":          updatedBy,
			})
		if result.Error != nil {
			if isForeignKeyViolation(result.Error) {
				return project.ErrInvalidStatusCode
			}
			return result.Error
		}
		if result.RowsAffected == 0 {
			return project.ErrProjectNotFound
		}
		return r.auditRepo.LogTx(tx, entry)
	})
	if err != nil {
		if errors.Is(err, project.ErrInvalidStatusCode) || errors.Is(err, project.ErrProjectNotFound) {
			return err
		}
		return fmt.Errorf("change project status: %w", err)
	}
	return nil
}

// isUniqueViolation detects Postgres unique constraint violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}

// isForeignKeyViolation detects Postgres foreign-key violation (SQLSTATE 23503).
func isForeignKeyViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23503")
}

// MasterRepo implements project.MasterRepository against Postgres via GORM.
type MasterRepo struct {
	db *gorm.DB
}

// NewMasterRepo constructs a MasterRepo holding only the base *gorm.DB.
func NewMasterRepo(db *gorm.DB) *MasterRepo {
	return &MasterRepo{db: db}
}

// IsActiveProjectStatusCode returns true when the code exists in project_statuses with status='active'.
func (r *MasterRepo) IsActiveProjectStatusCode(ctx context.Context, code string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).
		Table("project_statuses").
		Where("code = ? AND status = 'active'", code).
		Limit(1).
		Count(&n).Error
	if err != nil {
		return false, fmt.Errorf("check project_status_code: %w", err)
	}
	return n > 0, nil
}

// IsActiveProjectTypeCode returns true when the code exists in project_types with status='active'.
func (r *MasterRepo) IsActiveProjectTypeCode(ctx context.Context, code string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).
		Table("project_types").
		Where("code = ? AND status = 'active'", code).
		Limit(1).
		Count(&n).Error
	if err != nil {
		return false, fmt.Errorf("check project_type_code: %w", err)
	}
	return n > 0, nil
}
