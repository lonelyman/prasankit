package companypositiondbrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	auditdbrepo "prasankit-api/internal/adapters/database/audit"
	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/companyposition"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CompanyPositionRepo implements companyposition.CompanyPositionRepository against Postgres via GORM.
type CompanyPositionRepo struct {
	db        *gorm.DB
	auditRepo *auditdbrepo.AuditRepo
}

// NewCompanyPositionRepo constructs a CompanyPositionRepo.
func NewCompanyPositionRepo(db *gorm.DB, auditRepo *auditdbrepo.AuditRepo) *CompanyPositionRepo {
	return &CompanyPositionRepo{db: db, auditRepo: auditRepo}
}

// CreateWithAudit atomically inserts a company_position + audit_logs entry in one transaction.
// uq_company_positions_workspace_code violation -> ErrCodeTaken.
func (r *CompanyPositionRepo) CreateWithAudit(ctx context.Context, p companyposition.CompanyPosition, entry audit.Entry) error {
	model := toModel(p)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&model).Error; err != nil {
			if isUniqueViolation(err) {
				return companyposition.ErrCodeTaken
			}
			return err
		}
		return r.auditRepo.LogTx(tx, entry)
	})
	if err != nil {
		if errors.Is(err, companyposition.ErrCodeTaken) {
			return err
		}
		return fmt.Errorf("create company_position: %w", err)
	}
	return nil
}

// FindByCodeForWorkspace returns the row keyed on (workspace_id, code) covering active+deprecated.
func (r *CompanyPositionRepo) FindByCodeForWorkspace(ctx context.Context, workspaceID uuid.UUID, code string) (*companyposition.CompanyPosition, error) {
	var m companyPositionModel
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND code = ?", workspaceID, code).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find company_position: %w", err)
	}
	p := toDomain(m)
	return &p, nil
}

// ListByWorkspace returns the page of positions for the workspace (+ optional status filter).
func (r *CompanyPositionRepo) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, opts companyposition.ListOptions) ([]companyposition.CompanyPosition, int64, error) {
	tx := r.db.WithContext(ctx).Model(&companyPositionModel{}).
		Where("workspace_id = ?", workspaceID)
	if opts.Status != "" {
		tx = tx.Where("status = ?", opts.Status)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count company_positions: %w", err)
	}
	if total == 0 {
		return []companyposition.CompanyPosition{}, 0, nil
	}

	var rows []companyPositionModel
	offset := (opts.Page - 1) * opts.Limit
	if err := tx.Order("sort_order ASC, code ASC").Limit(opts.Limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list company_positions: %w", err)
	}

	out := make([]companyposition.CompanyPosition, 0, len(rows))
	for _, m := range rows {
		out = append(out, toDomain(m))
	}
	return out, total, nil
}

// UpdateWithAudit atomically updates the mutable fields + audit_logs entry in one transaction.
// The Updates map NEVER contains code/workspace_id/id/created_*.
func (r *CompanyPositionRepo) UpdateWithAudit(ctx context.Context, p companyposition.CompanyPosition, entry audit.Entry) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&companyPositionModel{}).
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
			return companyposition.ErrPositionNotFound
		}
		return r.auditRepo.LogTx(tx, entry)
	})
	if err != nil {
		if errors.Is(err, companyposition.ErrPositionNotFound) {
			return err
		}
		return fmt.Errorf("update company_position: %w", err)
	}
	return nil
}

// DeprecateWithAudit atomically sets status='deprecated' + audit_logs entry in one transaction.
func (r *CompanyPositionRepo) DeprecateWithAudit(ctx context.Context, workspaceID uuid.UUID, code string, updatedBy uuid.UUID, entry audit.Entry) error {
	now := time.Now().UTC()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&companyPositionModel{}).
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
			return companyposition.ErrPositionNotFound
		}
		return r.auditRepo.LogTx(tx, entry)
	})
	if err != nil {
		if errors.Is(err, companyposition.ErrPositionNotFound) {
			return err
		}
		return fmt.Errorf("deprecate company_position: %w", err)
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

// MasterRepo implements companyposition.CompanyPositionMasterRepository against Postgres via GORM.
// It binds workspace_id. Holds only *gorm.DB.
type MasterRepo struct {
	db *gorm.DB
}

// NewMasterRepo constructs a MasterRepo holding only the base *gorm.DB.
func NewMasterRepo(db *gorm.DB) *MasterRepo {
	return &MasterRepo{db: db}
}

// IsActiveCompanyPositionCode returns true when the code exists in company_positions
// for this workspace with status='active'.
func (r *MasterRepo) IsActiveCompanyPositionCode(ctx context.Context, workspaceID uuid.UUID, code string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).
		Table("company_positions").
		Where("workspace_id = ? AND code = ? AND status = 'active'", workspaceID, code).
		Limit(1).
		Count(&n).Error
	if err != nil {
		return false, fmt.Errorf("check company_position_code: %w", err)
	}
	return n > 0, nil
}
