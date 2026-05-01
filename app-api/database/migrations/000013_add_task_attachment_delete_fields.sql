-- +goose Up
ALTER TABLE task_attachments
    ADD COLUMN deleted_by UUID REFERENCES user_accounts(id),
    ADD COLUMN deleted_at TIMESTAMPTZ;

DROP INDEX IF EXISTS ix_task_attachments_task_created;
DROP INDEX IF EXISTS ix_task_attachments_status;

CREATE INDEX ix_task_attachments_task_created
    ON task_attachments (tenant_id, workspace_id, project_id, task_id, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX ix_task_attachments_status
    ON task_attachments (tenant_id, upload_status, created_at DESC)
    WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS ix_task_attachments_status;
DROP INDEX IF EXISTS ix_task_attachments_task_created;

CREATE INDEX ix_task_attachments_task_created
    ON task_attachments (tenant_id, workspace_id, project_id, task_id, created_at DESC);

CREATE INDEX ix_task_attachments_status
    ON task_attachments (tenant_id, upload_status, created_at DESC);

ALTER TABLE task_attachments
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS deleted_by;
