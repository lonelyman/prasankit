-- +goose Up
CREATE TABLE task_activities (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    project_id UUID NOT NULL REFERENCES projects(id),
    task_id UUID NOT NULL REFERENCES tasks(id),
    actor_account_id UUID NOT NULL REFERENCES user_accounts(id),
    action TEXT NOT NULL,
    from_status TEXT,
    to_status TEXT,
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ck_task_activities_action CHECK (
        action IN ('created', 'updated', 'status_changed', 'deleted')
    ),
    CONSTRAINT ck_task_activities_from_status CHECK (
        from_status IS NULL OR from_status IN ('todo', 'in_progress', 'blocked', 'done', 'cancelled')
    ),
    CONSTRAINT ck_task_activities_to_status CHECK (
        to_status IS NULL OR to_status IN ('todo', 'in_progress', 'blocked', 'done', 'cancelled')
    )
);

CREATE INDEX ix_task_activities_task_created
    ON task_activities (tenant_id, workspace_id, project_id, task_id, created_at DESC);

CREATE INDEX ix_task_activities_actor_created
    ON task_activities (tenant_id, actor_account_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS task_activities;
