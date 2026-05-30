-- +goose Up
-- D43: nullable, NO FK to projects(id) (audit append-only must outlive resource
-- deletion + breaks circular migration dependency).
ALTER TABLE audit_logs ADD COLUMN project_id UUID;

-- Invariant: a project-scoped audit row must also carry workspace_id (so the
-- D43 read invariant `WHERE workspace_id = ? AND project_id = ?` is total).
ALTER TABLE audit_logs
    ADD CONSTRAINT chk_audit_logs_project_implies_workspace
    CHECK (project_id IS NULL OR workspace_id IS NOT NULL);

CREATE INDEX ix_audit_workspace_project_created
    ON audit_logs (workspace_id, project_id, created_at DESC)
    WHERE project_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS ix_audit_workspace_project_created;
ALTER TABLE audit_logs DROP CONSTRAINT IF EXISTS chk_audit_logs_project_implies_workspace;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS project_id;
