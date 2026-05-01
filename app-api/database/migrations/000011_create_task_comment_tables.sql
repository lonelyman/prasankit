-- +goose Up
CREATE TABLE task_comments (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    project_id UUID NOT NULL REFERENCES projects(id),
    task_id UUID NOT NULL REFERENCES tasks(id),
    body TEXT NOT NULL,
    created_by UUID NOT NULL REFERENCES user_accounts(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by UUID REFERENCES user_accounts(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_by UUID REFERENCES user_accounts(id),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT ck_task_comments_body_not_blank CHECK (length(btrim(body)) > 0)
);

CREATE INDEX ix_task_comments_task_created
    ON task_comments (tenant_id, workspace_id, project_id, task_id, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX ix_task_comments_author_created
    ON task_comments (tenant_id, created_by, created_at DESC)
    WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS task_comments;
