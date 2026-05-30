// Package auditdbrepo implements the audit domain port against Postgres via GORM.
package auditdbrepo

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// auditLogModel is the GORM model for audit_logs.
type auditLogModel struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey"`
	WorkspaceID        *uuid.UUID `gorm:"type:uuid;column:workspace_id"`
	ProjectID          *uuid.UUID `gorm:"type:uuid;column:project_id"`
	ActorUserAccountID *uuid.UUID `gorm:"type:uuid;column:actor_user_account_id"`
	Action             string     `gorm:"column:action;not null"`
	ResourceType       string     `gorm:"column:resource_type;not null"`
	ResourceID         *uuid.UUID `gorm:"type:uuid;column:resource_id"`
	OldValue           []byte     `gorm:"column:old_value;type:jsonb"`
	NewValue           []byte     `gorm:"column:new_value;type:jsonb"`
	Result             string     `gorm:"column:result;not null"`
	IPAddress          string     `gorm:"column:ip_address;type:inet"`
	UserAgent          string     `gorm:"column:user_agent"`
	RequestID          string     `gorm:"column:request_id"`
	CreatedAt          time.Time  `gorm:"column:created_at;not null"`
}

func (auditLogModel) TableName() string { return "audit_logs" }

// marshalJSON converts a map to []byte for JSONB; returns nil for nil/empty maps.
func marshalJSON(m map[string]any) []byte {
	if len(m) == 0 {
		return nil
	}
	b, _ := json.Marshal(m)
	return b
}
