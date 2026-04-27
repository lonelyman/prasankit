-- +goose Up
CREATE TABLE user_accounts (
    id UUID PRIMARY KEY,
    primary_email CITEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending_verification',
    last_login_at TIMESTAMPTZ,
    failed_login_count INTEGER NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES user_accounts(id),
    CONSTRAINT ck_user_accounts_status CHECK (
        status IN ('pending_verification', 'active', 'suspended', 'disabled', 'deleted')
    ),
    CONSTRAINT ck_user_accounts_failed_login_count CHECK (failed_login_count >= 0)
);

CREATE INDEX ix_user_accounts_primary_email
    ON user_accounts (primary_email)
    WHERE deleted_at IS NULL;

CREATE INDEX ix_user_accounts_status
    ON user_accounts (status)
    WHERE deleted_at IS NULL;

CREATE TABLE auth_identities (
    id UUID PRIMARY KEY,
    user_account_id UUID NOT NULL REFERENCES user_accounts(id),
    identity_type TEXT NOT NULL,
    provider TEXT NOT NULL,
    provider_user_id TEXT,
    email CITEXT,
    email_verified_at TIMESTAMPTZ,
    password_hash TEXT,
    password_changed_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT ck_auth_identities_identity_type CHECK (
        identity_type IN ('email_password', 'oauth')
    ),
    CONSTRAINT ck_auth_identities_provider CHECK (
        provider IN ('email', 'google', 'facebook', 'microsoft', 'line', 'github')
    ),
    CONSTRAINT ck_auth_identities_email_password_shape CHECK (
        identity_type != 'email_password'
        OR (
            provider = 'email'
            AND provider_user_id IS NULL
            AND email IS NOT NULL
            AND password_hash IS NOT NULL
        )
    ),
    CONSTRAINT ck_auth_identities_oauth_shape CHECK (
        identity_type != 'oauth'
        OR (
            provider != 'email'
            AND provider_user_id IS NOT NULL
        )
    )
);

CREATE UNIQUE INDEX uq_auth_identities_provider_user_active
    ON auth_identities (provider, provider_user_id)
    WHERE provider_user_id IS NOT NULL AND deleted_at IS NULL;

CREATE UNIQUE INDEX uq_auth_identities_email_password_active
    ON auth_identities (provider, email)
    WHERE identity_type = 'email_password' AND provider = 'email' AND deleted_at IS NULL;

CREATE INDEX ix_auth_identities_user_account
    ON auth_identities (user_account_id)
    WHERE deleted_at IS NULL;

CREATE TABLE auth_sessions (
    id UUID PRIMARY KEY,
    user_account_id UUID NOT NULL REFERENCES user_accounts(id),
    session_key_hash TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    ip_address INET,
    user_agent TEXT,
    device_label TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    revoked_reason TEXT,
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    CONSTRAINT ck_auth_sessions_status CHECK (
        status IN ('active', 'revoked', 'expired')
    )
);

CREATE UNIQUE INDEX uq_auth_sessions_session_key_hash
    ON auth_sessions (session_key_hash);

CREATE INDEX ix_auth_sessions_user_status
    ON auth_sessions (user_account_id, status);

CREATE INDEX ix_auth_sessions_expires_at
    ON auth_sessions (expires_at);

CREATE TABLE auth_login_attempts (
    id UUID PRIMARY KEY,
    user_account_id UUID REFERENCES user_accounts(id),
    email CITEXT NOT NULL,
    success BOOLEAN NOT NULL,
    failure_reason TEXT,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ix_auth_login_attempts_email_created_at
    ON auth_login_attempts (email, created_at DESC);

CREATE INDEX ix_auth_login_attempts_user_created_at
    ON auth_login_attempts (user_account_id, created_at DESC);

CREATE TABLE auth_email_verification_tokens (
    id UUID PRIMARY KEY,
    auth_identity_id UUID NOT NULL REFERENCES auth_identities(id),
    token_hash TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    CONSTRAINT ck_auth_email_verification_tokens_status CHECK (
        status IN ('active', 'used', 'expired', 'revoked')
    )
);

CREATE UNIQUE INDEX uq_auth_email_verification_tokens_token_hash
    ON auth_email_verification_tokens (token_hash);

CREATE INDEX ix_auth_email_verification_tokens_identity_status
    ON auth_email_verification_tokens (auth_identity_id, status);

CREATE TABLE auth_password_reset_tokens (
    id UUID PRIMARY KEY,
    auth_identity_id UUID NOT NULL REFERENCES auth_identities(id),
    token_hash TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    CONSTRAINT ck_auth_password_reset_tokens_status CHECK (
        status IN ('active', 'used', 'expired', 'revoked')
    )
);

CREATE UNIQUE INDEX uq_auth_password_reset_tokens_token_hash
    ON auth_password_reset_tokens (token_hash);

CREATE INDEX ix_auth_password_reset_tokens_identity_status
    ON auth_password_reset_tokens (auth_identity_id, status);

CREATE TABLE security_events (
    id UUID PRIMARY KEY,
    user_account_id UUID REFERENCES user_accounts(id),
    event_type TEXT NOT NULL,
    severity TEXT NOT NULL DEFAULT 'info',
    ip_address INET,
    user_agent TEXT,
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ck_security_events_severity CHECK (
        severity IN ('info', 'warning', 'critical')
    )
);

CREATE INDEX ix_security_events_user_created_at
    ON security_events (user_account_id, created_at DESC);

CREATE INDEX ix_security_events_type_created_at
    ON security_events (event_type, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS security_events;
DROP TABLE IF EXISTS auth_password_reset_tokens;
DROP TABLE IF EXISTS auth_email_verification_tokens;
DROP TABLE IF EXISTS auth_login_attempts;
DROP TABLE IF EXISTS auth_sessions;
DROP TABLE IF EXISTS auth_identities;
DROP TABLE IF EXISTS user_accounts;
