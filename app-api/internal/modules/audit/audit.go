// Package audit defines the domain types and port interface for audit logging.
// It is pure — no gorm tags, no fiber imports.
package audit

import (
	"context"

	"github.com/google/uuid"
)

// Entry holds the data written to audit_logs.
type Entry struct {
	WorkspaceID    *uuid.UUID
	ActorAccountID *uuid.UUID
	Action         string
	ResourceType   string
	ResourceID     *uuid.UUID
	OldValue       map[string]any
	NewValue       map[string]any
	Result         string
	IP             string
	UserAgent      string
	RequestID      string
}

// Logger is the port for async/normal audit writes (outside a transaction).
type Logger interface {
	Log(ctx context.Context, entry Entry) error
}
