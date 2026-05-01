-- +goose Up
CREATE TABLE task_tags (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    project_id UUID NOT NULL REFERENCES projects(id),
    name TEXT NOT NULL,
    normalized_name TEXT NOT NULL,
    color TEXT,
    created_by UUID NOT NULL REFERENCES user_accounts(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by UUID REFERENCES user_accounts(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_by UUID REFERENCES user_accounts(id),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT ck_task_tags_name_not_blank CHECK (length(btrim(name)) > 0),
    CONSTRAINT ck_task_tags_normalized_name_not_blank CHECK (length(btrim(normalized_name)) > 0),
    CONSTRAINT ck_task_tags_color_not_blank CHECK (color IS NULL OR length(btrim(color)) > 0)
);

CREATE UNIQUE INDEX uq_task_tags_project_name_active
    ON task_tags (tenant_id, workspace_id, project_id, normalized_name)
    WHERE deleted_at IS NULL;

CREATE INDEX ix_task_tags_project_created
    ON task_tags (tenant_id, workspace_id, project_id, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE TABLE task_tag_assignments (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    project_id UUID NOT NULL REFERENCES projects(id),
    task_id UUID NOT NULL REFERENCES tasks(id),
    tag_id UUID NOT NULL REFERENCES task_tags(id),
    created_by UUID NOT NULL REFERENCES user_accounts(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_by UUID REFERENCES user_accounts(id),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX uq_task_tag_assignments_active
    ON task_tag_assignments (tenant_id, workspace_id, project_id, task_id, tag_id)
    WHERE deleted_at IS NULL;

CREATE INDEX ix_task_tag_assignments_task_created
    ON task_tag_assignments (tenant_id, workspace_id, project_id, task_id, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE TABLE task_relations (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    project_id UUID NOT NULL REFERENCES projects(id),
    source_task_id UUID NOT NULL REFERENCES tasks(id),
    target_task_id UUID NOT NULL REFERENCES tasks(id),
    relation_type TEXT NOT NULL,
    created_by UUID NOT NULL REFERENCES user_accounts(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_by UUID REFERENCES user_accounts(id),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT ck_task_relations_type CHECK (
        relation_type IN ('relates_to', 'blocks', 'blocked_by', 'duplicates')
    ),
    CONSTRAINT ck_task_relations_not_self CHECK (source_task_id <> target_task_id)
);

CREATE UNIQUE INDEX uq_task_relations_active
    ON task_relations (tenant_id, workspace_id, project_id, source_task_id, target_task_id, relation_type)
    WHERE deleted_at IS NULL;

CREATE INDEX ix_task_relations_source_created
    ON task_relations (tenant_id, workspace_id, project_id, source_task_id, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX ix_task_relations_target_created
    ON task_relations (tenant_id, workspace_id, project_id, target_task_id, created_at DESC)
    WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS task_relations;
DROP TABLE IF EXISTS task_tag_assignments;
DROP TABLE IF EXISTS task_tags;
