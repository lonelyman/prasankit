package auditdbrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"prasankit-api/internal/modules/audit"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
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
	if entry.ProjectID != nil && entry.WorkspaceID == nil {
		return fmt.Errorf("audit log: project_id set without workspace_id (D43 invariant)")
	}
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
	if entry.ProjectID != nil && entry.WorkspaceID == nil {
		return fmt.Errorf("audit log tx: project_id set without workspace_id (D43 invariant)")
	}
	m, err := entryToModel(entry)
	if err != nil {
		return fmt.Errorf("audit log tx: build model: %w", err)
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("audit log tx: insert: %w", err)
	}
	return nil
}

// ListByProject implements the D43 read invariant — caller binds workspace_id
// (from TenantContext) and project_id; the WHERE clause is total.
func (r *AuditRepo) ListByProject(ctx context.Context, workspaceID, projectID uuid.UUID) ([]audit.LogRow, error) {
	var rows []auditLogModel
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND project_id = ?", workspaceID, projectID).
		Order("created_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list audit by project: %w", err)
	}
	out := make([]audit.LogRow, 0, len(rows))
	for _, m := range rows {
		out = append(out, modelToLogRow(m))
	}
	return out, nil
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
		ProjectID:          entry.ProjectID,
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

// modelToLogRow converts the GORM model to the read-side LogRow.
func modelToLogRow(m auditLogModel) audit.LogRow {
	return audit.LogRow{
		ID:             m.ID,
		WorkspaceID:    m.WorkspaceID,
		ProjectID:      m.ProjectID,
		ActorAccountID: m.ActorUserAccountID,
		Action:         m.Action,
		ResourceType:   m.ResourceType,
		ResourceID:     m.ResourceID,
		Result:         m.Result,
		IPAddress:      m.IPAddress,
		UserAgent:      m.UserAgent,
		RequestID:      m.RequestID,
		OldValue:       unmarshalJSON(m.OldValue),
		NewValue:       unmarshalJSON(m.NewValue),
		CreatedAt:      m.CreatedAt,
	}
}

// unmarshalJSON converts a JSONB []byte payload back to a map. Returns nil if empty/invalid.
func unmarshalJSON(b []byte) map[string]any {
	if len(b) == 0 {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil
	}
	return out
}
