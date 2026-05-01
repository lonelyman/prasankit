-- +goose Up
CREATE TABLE task_views (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    project_id UUID NOT NULL REFERENCES projects(id),
    owner_account_id UUID NOT NULL REFERENCES user_accounts(id),
    name TEXT NOT NULL,
    normalized_name TEXT NOT NULL,
    filters_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_by UUID NOT NULL REFERENCES user_accounts(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by UUID REFERENCES user_accounts(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_by UUID REFERENCES user_accounts(id),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT ck_task_views_name_not_blank CHECK (length(btrim(name)) > 0),
    CONSTRAINT ck_task_views_normalized_name_not_blank CHECK (length(btrim(normalized_name)) > 0),
    CONSTRAINT ck_task_views_sort_order CHECK (sort_order >= 0)
);

CREATE UNIQUE INDEX uq_task_views_owner_name_active
    ON task_views (tenant_id, workspace_id, project_id, owner_account_id, normalized_name)
    WHERE deleted_at IS NULL;

CREATE INDEX ix_task_views_owner_sort
    ON task_views (tenant_id, workspace_id, project_id, owner_account_id, sort_order, created_at DESC)
    WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS task_views;
