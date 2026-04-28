-- +goose Up
CREATE TABLE workspace_roles (
    id UUID PRIMARY KEY,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_system BOOLEAN NOT NULL DEFAULT true,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_workspace_roles_code UNIQUE (code),
    CONSTRAINT ck_workspace_roles_code_format CHECK (
        code ~ '^[a-z][a-z0-9_]{1,62}$'
    ),
    CONSTRAINT ck_workspace_roles_status CHECK (
        status IN ('active', 'deprecated')
    ),
    CONSTRAINT ck_workspace_roles_sort_order CHECK (sort_order >= 0)
);

INSERT INTO workspace_roles (id, code, name, description, sort_order, is_system)
VALUES
    ('019dd667-fbfe-719e-bddb-4d3d5ecce176', 'owner', 'Owner', 'Workspace owner with full control', 10, true),
    ('019dd667-fbfe-723e-83de-b9472bab302a', 'admin', 'Admin', 'Workspace administrator', 20, true),
    ('019dd667-fbfe-7246-94a1-6d3489b71433', 'executive', 'Executive', 'Executive or manager viewer role', 30, true),
    ('019dd667-fbfe-724d-8ccf-7b021740eb08', 'user', 'User', 'Standard workspace user', 40, true)
ON CONFLICT (code) DO NOTHING;

ALTER TABLE workspace_memberships
    ADD COLUMN workspace_role_id UUID REFERENCES workspace_roles(id);

UPDATE workspace_memberships AS wm
SET workspace_role_id = wr.id
FROM workspace_roles AS wr
WHERE wr.code = wm.workspace_role;

ALTER TABLE workspace_memberships
    ALTER COLUMN workspace_role_id SET NOT NULL;

ALTER TABLE workspace_memberships
    DROP CONSTRAINT IF EXISTS ck_workspace_memberships_role;

ALTER TABLE workspace_memberships
    DROP COLUMN workspace_role;

CREATE INDEX ix_workspace_memberships_role_status
    ON workspace_memberships (workspace_role_id, status);

-- +goose Down
DROP INDEX IF EXISTS ix_workspace_memberships_role_status;

ALTER TABLE workspace_memberships
    ADD COLUMN workspace_role TEXT;

UPDATE workspace_memberships AS wm
SET workspace_role = wr.code
FROM workspace_roles AS wr
WHERE wr.id = wm.workspace_role_id;

ALTER TABLE workspace_memberships
    ALTER COLUMN workspace_role SET NOT NULL;

ALTER TABLE workspace_memberships
    ADD CONSTRAINT ck_workspace_memberships_role CHECK (
        workspace_role IN ('owner', 'admin', 'executive', 'user')
    );

ALTER TABLE workspace_memberships
    DROP COLUMN workspace_role_id;

DROP TABLE IF EXISTS workspace_roles;
