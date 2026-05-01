-- +goose Up
CREATE TABLE task_checklist_items (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    project_id UUID NOT NULL REFERENCES projects(id),
    task_id UUID NOT NULL REFERENCES tasks(id),
    item_text TEXT NOT NULL,
    is_completed BOOLEAN NOT NULL DEFAULT false,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_by UUID NOT NULL REFERENCES user_accounts(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by UUID REFERENCES user_accounts(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_by UUID REFERENCES user_accounts(id),
    completed_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES user_accounts(id),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT ck_task_checklist_items_text_not_blank CHECK (length(btrim(item_text)) > 0),
    CONSTRAINT ck_task_checklist_items_sort_order CHECK (sort_order >= 0),
    CONSTRAINT ck_task_checklist_items_completed_pair CHECK (
        (is_completed = false AND completed_at IS NULL AND completed_by IS NULL)
        OR
        (is_completed = true AND completed_at IS NOT NULL AND completed_by IS NOT NULL)
    )
);

CREATE INDEX ix_task_checklist_items_task_order
    ON task_checklist_items (tenant_id, workspace_id, project_id, task_id, sort_order ASC, created_at ASC)
    WHERE deleted_at IS NULL;

CREATE INDEX ix_task_checklist_items_completed
    ON task_checklist_items (tenant_id, is_completed, updated_at DESC)
    WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS task_checklist_items;
