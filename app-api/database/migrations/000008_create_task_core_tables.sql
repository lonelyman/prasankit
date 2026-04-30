-- +goose Up
CREATE TABLE task_counters (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    project_id UUID NOT NULL REFERENCES projects(id),
    prefix TEXT NOT NULL,
    running_no INTEGER NOT NULL DEFAULT 0,
    number_length INTEGER NOT NULL DEFAULT 4,
    last_generated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_task_counters_scope UNIQUE (tenant_id, workspace_id, project_id, prefix),
    CONSTRAINT ck_task_counters_running_no CHECK (running_no >= 0),
    CONSTRAINT ck_task_counters_number_length CHECK (number_length BETWEEN 3 AND 10)
);

CREATE TABLE tasks (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    project_id UUID NOT NULL REFERENCES projects(id),
    task_no TEXT NOT NULL,
    task_title TEXT NOT NULL,
    task_type_id UUID,
    status TEXT NOT NULL DEFAULT 'todo',
    priority_id UUID NOT NULL REFERENCES project_priorities(id),
    assignee_member_id UUID REFERENCES project_members(id),
    description TEXT,
    start_date DATE,
    due_date DATE,
    completed_date DATE,
    created_by UUID NOT NULL REFERENCES user_accounts(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by UUID REFERENCES user_accounts(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_by UUID REFERENCES user_accounts(id),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT ck_tasks_status CHECK (
        status IN ('todo', 'in_progress', 'blocked', 'done', 'cancelled')
    )
);

CREATE UNIQUE INDEX uq_tasks_no_active
    ON tasks (tenant_id, workspace_id, project_id, task_no)
    WHERE deleted_at IS NULL;

CREATE INDEX ix_tasks_project_status
    ON tasks (tenant_id, workspace_id, project_id, status)
    WHERE deleted_at IS NULL;

CREATE INDEX ix_tasks_project_created
    ON tasks (tenant_id, workspace_id, project_id, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX ix_tasks_assignee_status
    ON tasks (tenant_id, assignee_member_id, status)
    WHERE assignee_member_id IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX ix_tasks_due_date
    ON tasks (tenant_id, due_date)
    WHERE due_date IS NOT NULL AND deleted_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS task_counters;
