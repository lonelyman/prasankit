-- +goose Up
CREATE TABLE task_attachments (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    project_id UUID NOT NULL REFERENCES projects(id),
    task_id UUID NOT NULL REFERENCES tasks(id),
    file_name TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    storage_bucket TEXT NOT NULL,
    object_key TEXT NOT NULL,
    upload_status TEXT NOT NULL DEFAULT 'pending',
    uploaded_by UUID NOT NULL REFERENCES user_accounts(id),
    uploaded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_task_attachments_object_key UNIQUE (storage_bucket, object_key),
    CONSTRAINT ck_task_attachments_size_bytes CHECK (size_bytes > 0),
    CONSTRAINT ck_task_attachments_upload_status CHECK (
        upload_status IN ('pending', 'uploaded')
    )
);

CREATE INDEX ix_task_attachments_task_created
    ON task_attachments (tenant_id, workspace_id, project_id, task_id, created_at DESC);

CREATE INDEX ix_task_attachments_status
    ON task_attachments (tenant_id, upload_status, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS task_attachments;
