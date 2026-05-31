package projectpositiondbrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	auditdbrepo "prasankit-api/internal/adapters/database/audit"
	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/projectposition"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProjectPositionRepo implements projectposition.ProjectPositionRepository against Postgres via GORM.
type ProjectPositionRepo struct {
	db        *gorm.DB
	auditRepo *auditdbrepo.AuditRepo
}

// NewProjectPositionRepo constructs a ProjectPositionRepo.
func NewProjectPositionRepo(db *gorm.DB, auditRepo *auditdbrepo.AuditRepo) *ProjectPositionRepo {
	return &ProjectPositionRepo{db: db, auditRepo: auditRepo}
}

// CreateWithAudit atomically inserts a project_position + audit_logs entry in one transaction.
// uq_project_positions_workspace_code violation -> ErrCodeTaken.
func (r *ProjectPositionRepo) CreateWithAudit(ctx context.Context, p projectposition.ProjectPosition, entry audit.Entry) error {
	model := toModel(p)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&model).Error; err != nil {
			if isUniqueViolation(err) {
				return projectposition.ErrCodeTaken
			}
			return err
		}
		return r.auditRepo.LogTx(tx, entry)
	})
	if err != nil {
		if errors.Is(err, projectposition.ErrCodeTaken) {
			return err
		}
		return fmt.Errorf("create project_position: %w", err)
	}
	return nil
}

// FindByCodeForWorkspace returns the row keyed on (workspace_id, code) covering active+deprecated.
// gorm.ErrRecordNotFound -> (nil, nil). Other errors propagate.
func (r *ProjectPositionRepo) FindByCodeForWorkspace(ctx context.Context, workspaceID uuid.UUID, code string) (*projectposition.ProjectPosition, error) {
	var m projectPositionModel
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND code = ?", workspaceID, code).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find project_position: %w", err)
	}
	p := toDomain(m)
	return &p, nil
}

// ListByWorkspace returns the page of positions for the workspace (+ optional status filter).
func (r *ProjectPositionRepo) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, opts projectposition.ListOptions) ([]projectposition.ProjectPosition, int64, error) {
	tx := r.db.WithContext(ctx).Model(&projectPositionModel{}).
		Where("workspace_id = ?", workspaceID)
	if opts.Status != "" {
		tx = tx.Where("status = ?", opts.Status)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count project_positions: %w", err)
	}
	if total == 0 {
		return []projectposition.ProjectPosition{}, 0, nil
	}

	var rows []projectPositionModel
	offset := (opts.Page - 1) * opts.Limit
	if err := tx.Order("sort_order ASC, code ASC").Limit(opts.Limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list project_positions: %w", err)
	}

	out := make([]projectposition.ProjectPosition, 0, len(rows))
	for _, m := range rows {
		out = append(out, toDomain(m))
	}
	return out, total, nil
}

// UpdateWithAudit atomically updates the mutable fields + audit_logs entry in one transaction.
// The Updates map NEVER contains code/workspace_id/id/created_*.
func (r *ProjectPositionRepo) UpdateWithAudit(ctx context.Context, p projectposition.ProjectPosition, entry audit.Entry) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&projectPositionModel{}).
			Where("workspace_id = ? AND code = ?", p.WorkspaceID, p.Code).
			Updates(map[string]any{
				"label_th":    p.LabelTH,
				"label_en":    p.LabelEN,
				"description": p.Description,
				"sort_order":  p.SortOrder,
				"status":      p.Status,
				"updated_at":  p.UpdatedAt,
				"updated_by":  p.UpdatedBy,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return projectposition.ErrPositionNotFound
		}
		return r.auditRepo.LogTx(tx, entry)
	})
	if err != nil {
		if errors.Is(err, projectposition.ErrPositionNotFound) {
			return err
		}
		return fmt.Errorf("update project_position: %w", err)
	}
	return nil
}

// DeprecateWithAudit atomically sets status='deprecated' + audit_logs entry in one transaction.
func (r *ProjectPositionRepo) DeprecateWithAudit(ctx context.Context, workspaceID uuid.UUID, code string, updatedBy uuid.UUID, entry audit.Entry) error {
	now := time.Now().UTC()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&projectPositionModel{}).
			Where("workspace_id = ? AND code = ?", workspaceID, code).
			Updates(map[string]any{
				"status":     "deprecated",
				"updated_at": now,
				"updated_by": updatedBy,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return projectposition.ErrPositionNotFound
		}
		return r.auditRepo.LogTx(tx, entry)
	})
	if err != nil {
		if errors.Is(err, projectposition.ErrPositionNotFound) {
			return err
		}
		return fmt.Errorf("deprecate project_position: %w", err)
	}
	return nil
}

// isUniqueViolation detects Postgres unique constraint violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}

// MasterRepo implements projectposition.PositionMasterRepository against Postgres via GORM.
// It binds workspace_id (unlike the 6a global-master pre-check). Holds only *gorm.DB.
type MasterRepo struct {
	db *gorm.DB
}

// NewMasterRepo constructs a MasterRepo holding only the base *gorm.DB.
func NewMasterRepo(db *gorm.DB) *MasterRepo {
	return &MasterRepo{db: db}
}

// IsActiveProjectPositionCode returns true when the code exists in project_positions
// for this workspace with status='active'.
func (r *MasterRepo) IsActiveProjectPositionCode(ctx context.Context, workspaceID uuid.UUID, code string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).
		Table("project_positions").
		Where("workspace_id = ? AND code = ? AND status = 'active'", workspaceID, code).
		Limit(1).
		Count(&n).Error
	if err != nil {
		return false, fmt.Errorf("check project_position_code: %w", err)
	}
	return n > 0, nil
}
