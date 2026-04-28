-- +goose Up
CREATE TABLE workspaces (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    workspace_name TEXT NOT NULL,
    slug CITEXT NOT NULL,
    mode TEXT NOT NULL DEFAULT 'demo',
    status TEXT NOT NULL DEFAULT 'active',
    contact_email CITEXT NOT NULL,
    owner_user_account_id UUID NOT NULL REFERENCES user_accounts(id),
    email_verified_required BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by UUID NOT NULL REFERENCES user_accounts(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by UUID REFERENCES user_accounts(id),
    pending_deletion_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES user_accounts(id),
    hard_deleted_at TIMESTAMPTZ,
    CONSTRAINT uq_workspaces_tenant_id UNIQUE (tenant_id),
    CONSTRAINT ck_workspaces_mode CHECK (mode IN ('demo', 'production')),
    CONSTRAINT ck_workspaces_status CHECK (status IN ('active', 'suspended', 'pending_deletion', 'deleted')),
    CONSTRAINT ck_workspaces_slug_format CHECK (
        slug::text ~ '^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$'
    )
);

CREATE UNIQUE INDEX uq_workspaces_slug_active
    ON workspaces (slug)
    WHERE deleted_at IS NULL;

CREATE INDEX ix_workspaces_owner_status
    ON workspaces (owner_user_account_id, status)
    WHERE deleted_at IS NULL;

CREATE TABLE workspace_memberships (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    profile_id UUID,
    user_account_id UUID REFERENCES user_accounts(id),
    workspace_role TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    status_reason TEXT,
    joined_at TIMESTAMPTZ,
    removed_at TIMESTAMPTZ,
    suspended_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by UUID REFERENCES user_accounts(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by UUID REFERENCES user_accounts(id),
    CONSTRAINT ck_workspace_memberships_role CHECK (
        workspace_role IN ('owner', 'admin', 'executive', 'user')
    ),
    CONSTRAINT ck_workspace_memberships_status CHECK (
        status IN ('active', 'removed', 'suspended')
    ),
    CONSTRAINT ck_workspace_memberships_tenant_workspace CHECK (tenant_id IS NOT NULL AND workspace_id IS NOT NULL)
);

CREATE INDEX ix_workspace_memberships_workspace_status
    ON workspace_memberships (tenant_id, workspace_id, status);

CREATE INDEX ix_workspace_memberships_user_status
    ON workspace_memberships (user_account_id, status)
    WHERE user_account_id IS NOT NULL;

CREATE UNIQUE INDEX uq_workspace_memberships_active_user
    ON workspace_memberships (tenant_id, user_account_id)
    WHERE user_account_id IS NOT NULL AND status = 'active';

-- +goose Down
DROP TABLE IF EXISTS workspace_memberships;
DROP TABLE IF EXISTS workspaces;
