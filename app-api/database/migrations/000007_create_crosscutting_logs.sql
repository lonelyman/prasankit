-- +goose Up

CREATE TABLE audit_logs (
    id                    UUID        PRIMARY KEY,
    workspace_id          UUID        REFERENCES workspaces(id),
    actor_user_account_id UUID        REFERENCES user_accounts(id),
    action                TEXT        NOT NULL,
    resource_type         TEXT        NOT NULL,
    resource_id           UUID,
    old_value             JSONB,
    new_value             JSONB,
    result                TEXT        NOT NULL,
    ip_address            INET,
    user_agent            TEXT,
    request_id            TEXT,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ix_audit_workspace_created
    ON audit_logs (workspace_id, created_at DESC);

CREATE INDEX ix_audit_actor_created
    ON audit_logs (actor_user_account_id, created_at DESC);

CREATE TABLE security_events (
    id               UUID        PRIMARY KEY,
    user_account_id  UUID        REFERENCES user_accounts(id),
    event_type       TEXT        NOT NULL,
    severity         TEXT        NOT NULL DEFAULT 'info',
    ip_address       INET,
    user_agent       TEXT,
    metadata         JSONB       NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ix_security_user_created
    ON security_events (user_account_id, created_at DESC);

CREATE INDEX ix_security_type_created
    ON security_events (event_type, created_at DESC);

-- +goose Down

DROP TABLE IF EXISTS security_events;
DROP TABLE IF EXISTS audit_logs;
