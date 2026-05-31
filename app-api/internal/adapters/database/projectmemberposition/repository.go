package projectmemberpositiondbrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	auditdbrepo "prasankit-api/internal/adapters/database/audit"
	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/projectmemberposition"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProjectMemberPositionRepo implements projectmemberposition.ProjectMemberPositionRepository
// against Postgres via GORM. It does NOT import projectmemberdbrepo or any position adapter
// (adapter->adapter forbidden, 02 §3) — the JOIN is done in-adapter against project_positions.
type ProjectMemberPositionRepo struct {
	db        *gorm.DB
	auditRepo *auditdbrepo.AuditRepo
}

// NewProjectMemberPositionRepo constructs a ProjectMemberPositionRepo.
func NewProjectMemberPositionRepo(db *gorm.DB, auditRepo *auditdbrepo.AuditRepo) *ProjectMemberPositionRepo {
	return &ProjectMemberPositionRepo{db: db, auditRepo: auditRepo}
}

// AssignWithAudit atomically inserts a junction row + audit_logs entry in one transaction.
// uq_pmp_member_position violation -> ErrAlreadyAssigned. Any 23503 -> ErrInvalidPositionCode
// (the member composite FK is service-pre-guarded; the adapter does NOT distinguish the two FKs).
func (r *ProjectMemberPositionRepo) AssignWithAudit(ctx context.Context, mp projectmemberposition.ProjectMemberPosition, entry audit.Entry) error {
	model := toModel(mp)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&model).Error; err != nil {
			if isUniqueViolation(err) {
				return projectmemberposition.ErrAlreadyAssigned
			}
			if isForeignKeyViolation(err) {
				return projectmemberposition.ErrInvalidPositionCode
			}
			return err
		}
		return r.auditRepo.LogTx(tx, entry)
	})
	if err != nil {
		if errors.Is(err, projectmemberposition.ErrAlreadyAssigned) || errors.Is(err, projectmemberposition.ErrInvalidPositionCode) {
			return err
		}
		return fmt.Errorf("assign project_member_position: %w", err)
	}
	return nil
}

// UnassignWithAudit atomically hard-DELETEs the junction row + audit_logs entry in one transaction.
// RowsAffected==0 -> ErrAssignmentNotFound. The audit row is written in the same tx (D31).
func (r *ProjectMemberPositionRepo) UnassignWithAudit(ctx context.Context, workspaceID, projectMemberID uuid.UUID, positionCode string, entry audit.Entry) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.
			Where("workspace_id = ? AND project_member_id = ? AND project_position_code = ?", workspaceID, projectMemberID, positionCode).
			Delete(&projectMemberPositionModel{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return projectmemberposition.ErrAssignmentNotFound
		}
		return r.auditRepo.LogTx(tx, entry)
	})
	if err != nil {
		if errors.Is(err, projectmemberposition.ErrAssignmentNotFound) {
			return err
		}
		return fmt.Errorf("unassign project_member_position: %w", err)
	}
	return nil
}

// ListByMember returns the member's positions, JOINed to project_positions (same-ws), with labels.
// The pp.workspace_id = pmp.workspace_id JOIN predicate is load-bearing: it prevents a cross-ws
// label leak when the same code exists in two workspaces (mirror projectmember's JOIN predicate).
func (r *ProjectMemberPositionRepo) ListByMember(ctx context.Context, workspaceID, projectMemberID uuid.UUID) ([]projectmemberposition.PositionWithLabel, error) {
	var rows []positionWithLabelRow
	err := r.db.WithContext(ctx).
		Table("project_member_positions AS pmp").
		Select(`pmp.id, pmp.workspace_id, pmp.project_member_id, pmp.project_position_code,
			pmp.created_at, pmp.created_by, pp.label_th AS label_th, pp.label_en AS label_en,
			pp.status AS status`).
		Joins("INNER JOIN project_positions pp ON pp.workspace_id = pmp.workspace_id AND pp.code = pmp.project_position_code").
		Where("pmp.workspace_id = ? AND pmp.project_member_id = ?", workspaceID, projectMemberID).
		Order("pmp.created_at ASC").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list project_member_positions: %w", err)
	}
	out := make([]projectmemberposition.PositionWithLabel, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToPositionWithLabel(row))
	}
	return out, nil
}

// Compile-time assertion that the adapter satisfies the domain port.
var _ projectmemberposition.ProjectMemberPositionRepository = (*ProjectMemberPositionRepo)(nil)

// isUniqueViolation detects Postgres unique constraint violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}

// isForeignKeyViolation detects Postgres foreign-key violation (SQLSTATE 23503).
func isForeignKeyViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23503")
}
