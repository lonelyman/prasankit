package companypositiondbrepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	auditdbrepo "prasankit-api/internal/adapters/database/audit"
	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/companyposition"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// membershipRow is the file-private GORM model used to UPDATE workspace_memberships
// .company_position_code (D41 attach). It lives here (NOT in the workspace adapter) so this
// adapter does not import another adapter package (hexagonal, 02 §3) — the proven ownerMemberRow
// escape hatch. TableName() is REQUIRED: without it GORM infers "membership_rows".
// It is read-only for the prior value + write-only for {company_position_code,updated_at,updated_by}.
type membershipRow struct {
	ID                  uuid.UUID `gorm:"column:id;primaryKey"`
	WorkspaceID         uuid.UUID `gorm:"column:workspace_id"`
	CompanyPositionCode *string   `gorm:"column:company_position_code"`
}

func (membershipRow) TableName() string { return "workspace_memberships" }

// MembershipPositionRepo implements companyposition.MembershipPositionRepository against
// Postgres via GORM. It does NOT import workspacedbrepo.
type MembershipPositionRepo struct {
	db        *gorm.DB
	auditRepo *auditdbrepo.AuditRepo
}

// NewMembershipPositionRepo constructs a MembershipPositionRepo.
func NewMembershipPositionRepo(db *gorm.DB, auditRepo *auditdbrepo.AuditRepo) *MembershipPositionRepo {
	return &MembershipPositionRepo{db: db, auditRepo: auditRepo}
}

// SetCompanyPositionWithAudit atomically UPDATEs the membership's company_position_code +
// audit_logs entry in one transaction, gating on membership_status_code='active' (mirrors
// projectmember.IsActiveWorkspaceMember). The WHERE binds workspace_id so a cross-ws membership
// id yields RowsAffected==0 -> ErrMembershipNotFound (isolation, §4.3 invariant #3). The Updates
// map touches ONLY {company_position_code, updated_at, updated_by}. 23503 -> ErrInvalidPositionCode
// (FK backstop). The entry's OldValue is populated in-tx from the affected row's prior value.
func (r *MembershipPositionRepo) SetCompanyPositionWithAudit(ctx context.Context, workspaceID, membershipID uuid.UUID, code *string, updatedBy uuid.UUID, entry audit.Entry) error {
	now := time.Now().UTC()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Read the prior company_position_code for the audit OldValue (scoped + active-gated,
		// so a not-found/inactive row yields nil here and RowsAffected==0 on the UPDATE below).
		var prior membershipRow
		readErr := tx.Model(&membershipRow{}).
			Where("workspace_id = ? AND id = ? AND membership_status_code = 'active'", workspaceID, membershipID).
			First(&prior).Error
		if readErr != nil && !errors.Is(readErr, gorm.ErrRecordNotFound) {
			return readErr
		}
		if readErr == nil && prior.CompanyPositionCode != nil {
			entry.OldValue = map[string]any{"company_position_code": *prior.CompanyPositionCode}
		}

		result := tx.Model(&membershipRow{}).
			Where("workspace_id = ? AND id = ? AND membership_status_code = 'active'", workspaceID, membershipID).
			Updates(map[string]any{
				"company_position_code": code,
				"updated_at":            now,
				"updated_by":            updatedBy,
			})
		if result.Error != nil {
			if isForeignKeyViolation(result.Error) {
				return companyposition.ErrInvalidPositionCode
			}
			return result.Error
		}
		if result.RowsAffected == 0 {
			return companyposition.ErrMembershipNotFound
		}
		return r.auditRepo.LogTx(tx, entry)
	})
	if err != nil {
		if errors.Is(err, companyposition.ErrInvalidPositionCode) || errors.Is(err, companyposition.ErrMembershipNotFound) {
			return err
		}
		return fmt.Errorf("set membership company_position: %w", err)
	}
	return nil
}
