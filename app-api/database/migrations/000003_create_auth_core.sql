-- +goose Up

CREATE TABLE user_accounts (
    id                   UUID        PRIMARY KEY,
    primary_email        CITEXT      NOT NULL,
    display_name         TEXT        NOT NULL,
    account_status_code  TEXT        NOT NULL REFERENCES account_statuses(code),
    failed_login_count   INT         NOT NULL DEFAULT 0,
    locked_until         TIMESTAMPTZ,
    last_login_at        TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at           TIMESTAMPTZ,
    deleted_by           UUID        REFERENCES user_accounts(id),
    CONSTRAINT ck_user_accounts_failed_login_count CHECK (failed_login_count >= 0)
);

CREATE UNIQUE INDEX uq_user_accounts_primary_email_active
    ON user_accounts (primary_email)
    WHERE deleted_at IS NULL;

CREATE INDEX ix_user_accounts_account_status
    ON user_accounts (account_status_code)
    WHERE deleted_at IS NULL;

CREATE TABLE auth_identities (
    id                   UUID        PRIMARY KEY,
    user_account_id      UUID        NOT NULL REFERENCES user_accounts(id),
    identity_type_code   TEXT        NOT NULL REFERENCES auth_identity_types(code),
    email                CITEXT,
    email_verified_at    TIMESTAMPTZ,
    password_hash        TEXT,
    password_changed_at  TIMESTAMPTZ,
    last_used_at         TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at           TIMESTAMPTZ
);

CREATE UNIQUE INDEX uq_auth_identities_email_password_active
    ON auth_identities (email)
    WHERE email IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX ix_auth_identities_user_account
    ON auth_identities (user_account_id)
    WHERE deleted_at IS NULL;

CREATE TABLE auth_email_verification_tokens (
    id                UUID        PRIMARY KEY,
    auth_identity_id  UUID        NOT NULL REFERENCES auth_identities(id),
    token_hash        TEXT        NOT NULL,
    expires_at        TIMESTAMPTZ NOT NULL,
    used_at           TIMESTAMPTZ,
    revoked_at        TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_email_verif_token_hash UNIQUE (token_hash)
);

CREATE INDEX ix_email_verif_identity
    ON auth_email_verification_tokens (auth_identity_id);

CREATE TABLE auth_password_reset_tokens (
    id                UUID        PRIMARY KEY,
    auth_identity_id  UUID        NOT NULL REFERENCES auth_identities(id),
    token_hash        TEXT        NOT NULL,
    expires_at        TIMESTAMPTZ NOT NULL,
    used_at           TIMESTAMPTZ,
    revoked_at        TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_pwreset_token_hash UNIQUE (token_hash)
);

CREATE INDEX ix_pwreset_identity
    ON auth_password_reset_tokens (auth_identity_id);

-- +goose Down

DROP TABLE IF EXISTS auth_password_reset_tokens;
DROP TABLE IF EXISTS auth_email_verification_tokens;
DROP TABLE IF EXISTS auth_identities;
DROP TABLE IF EXISTS user_accounts;
