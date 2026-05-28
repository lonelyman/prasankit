package auditdbrepo

import (
	"context"
	"fmt"
	"time"

	"prasankit-api/internal/modules/audit"
	"prasankit-api/pkg/ids"

	"gorm.io/gorm"
)

// AuditRepo implements audit.Logger against Postgres via GORM.
// It also exposes LogTx for use inside another repo's transaction.
type AuditRepo struct {
	db *gorm.DB
}

// NewAuditRepo constructs an AuditRepo holding only the base *gorm.DB.
func NewAuditRepo(db *gorm.DB) *AuditRepo {
	return &AuditRepo{db: db}
}

// Log writes an audit entry outside any existing transaction.
// Implements audit.Logger.
func (r *AuditRepo) Log(ctx context.Context, entry audit.Entry) error {
	m, err := entryToModel(entry)
	if err != nil {
		return fmt.Errorf("audit log: build model: %w", err)
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("audit log: insert: %w", err)
	}
	return nil
}

// LogTx writes an audit entry inside an already-open GORM transaction.
// The workspace repo calls this to keep audit_logs knowledge inside this adapter.
func (r *AuditRepo) LogTx(tx *gorm.DB, entry audit.Entry) error {
	m, err := entryToModel(entry)
	if err != nil {
		return fmt.Errorf("audit log tx: build model: %w", err)
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("audit log tx: insert: %w", err)
	}
	return nil
}

// entryToModel converts an audit.Entry to the GORM model, generating the PK.
func entryToModel(entry audit.Entry) (auditLogModel, error) {
	id, err := ids.New()
	if err != nil {
		return auditLogModel{}, fmt.Errorf("generate audit log id: %w", err)
	}
	return auditLogModel{
		ID:                 id,
		WorkspaceID:        entry.WorkspaceID,
		ActorUserAccountID: entry.ActorAccountID,
		Action:             entry.Action,
		ResourceType:       entry.ResourceType,
		ResourceID:         entry.ResourceID,
		OldValue:           marshalJSON(entry.OldValue),
		NewValue:           marshalJSON(entry.NewValue),
		Result:             entry.Result,
		IPAddress:          entry.IP,
		UserAgent:          entry.UserAgent,
		RequestID:          entry.RequestID,
		CreatedAt:          time.Now().UTC(),
	}, nil
}
