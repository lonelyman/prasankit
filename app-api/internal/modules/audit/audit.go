// Package audit defines the domain types and port interface for audit logging.
// It is pure — no gorm tags, no fiber imports.
package audit

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Entry holds the data written to audit_logs.
//
// ProjectID was added for D43 invariant #6: any project-scoped audit row MUST
// also carry workspace_id. The repo (LogTx / Log) asserts at write time that
// ProjectID != nil implies WorkspaceID != nil.
type Entry struct {
	WorkspaceID    *uuid.UUID
	ProjectID      *uuid.UUID // D43 invariant #6: non-nil implies WorkspaceID non-nil
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

// LogRow is the read-side representation of an audit_logs row (separate from
// Entry, which is the write-side shape). Used by ListByProject for the D43
// read-invariant proof test.
type LogRow struct {
	ID             uuid.UUID
	WorkspaceID    *uuid.UUID
	ProjectID      *uuid.UUID
	ActorAccountID *uuid.UUID
	Action         string
	ResourceType   string
	ResourceID     *uuid.UUID
	Result         string
	IPAddress      string
	UserAgent      string
	RequestID      string
	OldValue       map[string]any
	NewValue       map[string]any
	CreatedAt      time.Time
}

// Logger is the port for async/normal audit writes (outside a transaction).
type Logger interface {
	Log(ctx context.Context, entry Entry) error
}
