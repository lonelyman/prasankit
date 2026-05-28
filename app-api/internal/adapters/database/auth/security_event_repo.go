package authdbrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"prasankit-api/internal/modules/auth"

	"gorm.io/gorm"
)

// SecurityEventRepo implements auth.SecurityEventRepository against Postgres via GORM.
type SecurityEventRepo struct {
	db *gorm.DB
}

// NewSecurityEventRepo constructs a SecurityEventRepo holding only the base *gorm.DB.
func NewSecurityEventRepo(db *gorm.DB) *SecurityEventRepo {
	return &SecurityEventRepo{db: db}
}

// Log appends a security event row. Failures must not abort the caller.
func (r *SecurityEventRepo) Log(ctx context.Context, event auth.SecurityEvent) error {
	meta, err := json.Marshal(event.Metadata)
	if err != nil {
		meta = []byte("{}")
	}

	m := securityEventModel{
		ID:            event.ID,
		UserAccountID: event.UserAccountID,
		EventType:     event.EventType,
		Severity:      event.Severity,
		IPAddress:     event.IPAddress,
		UserAgent:     event.UserAgent,
		Metadata:      string(meta),
		CreatedAt:     time.Now().UTC(),
	}

	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("log security event: %w", err)
	}
	return nil
}
