-- +goose Up
CREATE TABLE project_roles (
    id UUID PRIMARY KEY,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_system BOOLEAN NOT NULL DEFAULT true,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_project_roles_code UNIQUE (code),
    CONSTRAINT ck_project_roles_code_format CHECK (
        code ~ '^[a-z][a-z0-9_]{1,62}$'
    ),
    CONSTRAINT ck_project_roles_status CHECK (
        status IN ('active', 'deprecated')
    ),
    CONSTRAINT ck_project_roles_sort_order CHECK (sort_order >= 0)
);

INSERT INTO project_roles (id, code, name, description, sort_order, is_system)
VALUES
    ('019dd690-f802-7591-ace9-bd02bc1e3687', 'project_owner', 'Project Owner', 'Project owner with full control', 10, true),
    ('019dd690-f802-7668-8d0b-2d8b507dd3ab', 'project_manager', 'Project Manager', 'Project manager', 20, true),
    ('019dd690-f802-7674-bc45-11e6c94dc93b', 'member', 'Member', 'Standard project member', 30, true),
    ('019dd690-f802-767c-9af0-f8293501ce1b', 'finance', 'Finance', 'Finance project member', 40, true),
    ('019dd690-f802-7683-9b46-0a91375a2c22', 'viewer', 'Viewer', 'Read-only project viewer', 50, true);

CREATE TABLE project_priorities (
    id UUID PRIMARY KEY,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_system BOOLEAN NOT NULL DEFAULT true,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_project_priorities_code UNIQUE (code),
    CONSTRAINT ck_project_priorities_code_format CHECK (
        code ~ '^[a-z][a-z0-9_]{1,62}$'
    ),
    CONSTRAINT ck_project_priorities_status CHECK (
        status IN ('active', 'deprecated')
    ),
    CONSTRAINT ck_project_priorities_sort_order CHECK (sort_order >= 0)
);

INSERT INTO project_priorities (id, code, name, sort_order, is_system)
VALUES
    ('019dd691-3650-77c8-8634-debe5ef4c4f8', 'low', 'Low', 10, true),
    ('019dd691-3650-7835-af1a-f3c8ad77ced5', 'medium', 'Medium', 20, true),
    ('019dd691-3650-7839-8230-012456f299ac', 'high', 'High', 30, true);

CREATE TABLE project_code_counters (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    prefix TEXT NOT NULL,
    year INTEGER NOT NULL,
    running_no INTEGER NOT NULL DEFAULT 0,
    number_length INTEGER NOT NULL DEFAULT 4,
    last_generated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_project_code_counters_scope UNIQUE (tenant_id, workspace_id, prefix, year),
    CONSTRAINT ck_project_code_counters_year CHECK (year >= 2000),
    CONSTRAINT ck_project_code_counters_running_no CHECK (running_no >= 0),
    CONSTRAINT ck_project_code_counters_number_length CHECK (number_length BETWEEN 3 AND 10)
);

CREATE TABLE projects (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    project_code TEXT NOT NULL,
    project_name TEXT NOT NULL,
    project_type TEXT NOT NULL DEFAULT 'internal',
    project_status TEXT NOT NULL DEFAULT 'draft',
    priority_id UUID NOT NULL REFERENCES project_priorities(id),
    description TEXT,
    client_or_requesting_unit TEXT,
    scope_or_objective TEXT,
    created_by UUID NOT NULL REFERENCES user_accounts(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by UUID REFERENCES user_accounts(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_by UUID REFERENCES user_accounts(id),
    deleted_at TIMESTAMPTZ,
    archived_at TIMESTAMPTZ,
    CONSTRAINT ck_projects_type CHECK (project_type IN ('internal', 'client')),
    CONSTRAINT ck_projects_status CHECK (
        project_status IN ('draft', 'planning', 'proposal', 'active', 'closing', 'maintenance', 'closed', 'archived')
    )
);

CREATE UNIQUE INDEX uq_projects_code_active
    ON projects (tenant_id, workspace_id, project_code)
    WHERE deleted_at IS NULL;

CREATE INDEX ix_projects_workspace_status
    ON projects (tenant_id, workspace_id, project_status)
    WHERE deleted_at IS NULL;

CREATE INDEX ix_projects_workspace_created
    ON projects (tenant_id, workspace_id, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE TABLE project_members (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    project_id UUID NOT NULL REFERENCES projects(id),
    workspace_membership_id UUID NOT NULL REFERENCES workspace_memberships(id),
    profile_id UUID,
    user_account_id UUID REFERENCES user_accounts(id),
    project_role_id UUID NOT NULL REFERENCES project_roles(id),
    status TEXT NOT NULL DEFAULT 'active',
    joined_at TIMESTAMPTZ,
    removed_at TIMESTAMPTZ,
    created_by UUID REFERENCES user_accounts(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by UUID REFERENCES user_accounts(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ck_project_members_status CHECK (status IN ('active', 'removed'))
);

CREATE UNIQUE INDEX uq_project_members_active_workspace_membership
    ON project_members (tenant_id, project_id, workspace_membership_id)
    WHERE status = 'active';

CREATE INDEX ix_project_members_project_status
    ON project_members (tenant_id, project_id, status);

CREATE INDEX ix_project_members_user_status
    ON project_members (tenant_id, user_account_id, status)
    WHERE user_account_id IS NOT NULL;

CREATE INDEX ix_project_members_role_status
    ON project_members (project_role_id, status);

-- +goose Down
DROP TABLE IF EXISTS project_members;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS project_code_counters;
DROP TABLE IF EXISTS project_priorities;
DROP TABLE IF EXISTS project_roles;
