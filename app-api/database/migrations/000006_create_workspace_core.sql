-- +goose Up

CREATE TABLE workspaces (
    id                     UUID        PRIMARY KEY,
    workspace_name         TEXT        NOT NULL,
    slug                   CITEXT      NOT NULL,
    workspace_status_code  TEXT        NOT NULL REFERENCES workspace_statuses(code),
    contact_email          CITEXT      NOT NULL,
    owner_user_account_id  UUID        NOT NULL REFERENCES user_accounts(id),
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by             UUID        REFERENCES user_accounts(id),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by             UUID        REFERENCES user_accounts(id),
    pending_deletion_at    TIMESTAMPTZ,
    deleted_at             TIMESTAMPTZ,
    deleted_by             UUID        REFERENCES user_accounts(id),
    CONSTRAINT ck_workspaces_slug CHECK (slug::text ~ '^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$')
);

CREATE UNIQUE INDEX uq_workspaces_slug_active
    ON workspaces (slug)
    WHERE deleted_at IS NULL;

CREATE INDEX ix_workspaces_owner_status
    ON workspaces (owner_user_account_id, workspace_status_code)
    WHERE deleted_at IS NULL;

CREATE TABLE workspace_memberships (
    id                          UUID        PRIMARY KEY,
    workspace_id                UUID        NOT NULL REFERENCES workspaces(id),
    user_account_id             UUID        NOT NULL REFERENCES user_accounts(id),
    org_role_code               TEXT        NOT NULL REFERENCES org_roles(code),
    membership_status_code      TEXT        NOT NULL REFERENCES membership_statuses(code),
    invited_by_user_account_id  UUID        REFERENCES user_accounts(id),
    joined_at                   TIMESTAMPTZ,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by                  UUID        REFERENCES user_accounts(id),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by                  UUID        REFERENCES user_accounts(id)
);

CREATE UNIQUE INDEX uq_membership_active_user
    ON workspace_memberships (workspace_id, user_account_id)
    WHERE membership_status_code = 'active';

CREATE INDEX ix_membership_workspace_status
    ON workspace_memberships (workspace_id, membership_status_code);

CREATE INDEX ix_membership_user
    ON workspace_memberships (user_account_id);

CREATE TABLE workspace_invitations (
    id                          UUID        PRIMARY KEY,
    workspace_id                UUID        NOT NULL REFERENCES workspaces(id),
    email                       CITEXT      NOT NULL,
    org_role_code               TEXT        NOT NULL REFERENCES org_roles(code),
    token_hash                  TEXT        NOT NULL,
    invited_by_user_account_id  UUID        NOT NULL REFERENCES user_accounts(id),
    expires_at                  TIMESTAMPTZ NOT NULL,
    accepted_at                 TIMESTAMPTZ,
    revoked_at                  TIMESTAMPTZ,
    accepted_user_account_id    UUID        REFERENCES user_accounts(id),
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_invitation_token_hash UNIQUE (token_hash)
);

CREATE UNIQUE INDEX uq_invitation_pending
    ON workspace_invitations (workspace_id, email)
    WHERE accepted_at IS NULL AND revoked_at IS NULL;

CREATE INDEX ix_invitation_workspace
    ON workspace_invitations (workspace_id);

-- +goose Down

DROP TABLE IF EXISTS workspace_invitations;
DROP TABLE IF EXISTS workspace_memberships;
DROP TABLE IF EXISTS workspaces;
