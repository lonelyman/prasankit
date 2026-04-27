-- +goose Up
DROP INDEX IF EXISTS uq_user_accounts_email_active;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'user_accounts'
          AND column_name = 'email'
    ) THEN
        ALTER TABLE user_accounts RENAME COLUMN email TO primary_email;
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE user_accounts DROP COLUMN IF EXISTS password_hash;
ALTER TABLE user_accounts DROP COLUMN IF EXISTS email_verified_at;
ALTER TABLE user_accounts DROP COLUMN IF EXISTS password_changed_at;

CREATE INDEX IF NOT EXISTS ix_user_accounts_primary_email
    ON user_accounts (primary_email)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS auth_identities (
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

CREATE UNIQUE INDEX IF NOT EXISTS uq_auth_identities_provider_user_active
    ON auth_identities (provider, provider_user_id)
    WHERE provider_user_id IS NOT NULL AND deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_auth_identities_email_password_active
    ON auth_identities (provider, email)
    WHERE identity_type = 'email_password' AND provider = 'email' AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_auth_identities_user_account
    ON auth_identities (user_account_id)
    WHERE deleted_at IS NULL;

ALTER TABLE auth_email_verification_tokens
    ADD COLUMN IF NOT EXISTS auth_identity_id UUID REFERENCES auth_identities(id);

ALTER TABLE auth_password_reset_tokens
    ADD COLUMN IF NOT EXISTS auth_identity_id UUID REFERENCES auth_identities(id);

ALTER TABLE auth_email_verification_tokens DROP COLUMN IF EXISTS user_account_id;
ALTER TABLE auth_password_reset_tokens DROP COLUMN IF EXISTS user_account_id;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'auth_email_verification_tokens'
          AND column_name = 'auth_identity_id'
          AND is_nullable = 'YES'
    ) THEN
        ALTER TABLE auth_email_verification_tokens ALTER COLUMN auth_identity_id SET NOT NULL;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'auth_password_reset_tokens'
          AND column_name = 'auth_identity_id'
          AND is_nullable = 'YES'
    ) THEN
        ALTER TABLE auth_password_reset_tokens ALTER COLUMN auth_identity_id SET NOT NULL;
    END IF;
END $$;
-- +goose StatementEnd

DROP INDEX IF EXISTS ix_auth_email_verification_tokens_user_status;
DROP INDEX IF EXISTS ix_auth_password_reset_tokens_user_status;

CREATE INDEX IF NOT EXISTS ix_auth_email_verification_tokens_identity_status
    ON auth_email_verification_tokens (auth_identity_id, status);

CREATE INDEX IF NOT EXISTS ix_auth_password_reset_tokens_identity_status
    ON auth_password_reset_tokens (auth_identity_id, status);

-- +goose Down
DROP INDEX IF EXISTS ix_auth_password_reset_tokens_identity_status;
DROP INDEX IF EXISTS ix_auth_email_verification_tokens_identity_status;

ALTER TABLE auth_password_reset_tokens
    ADD COLUMN IF NOT EXISTS user_account_id UUID REFERENCES user_accounts(id);

ALTER TABLE auth_email_verification_tokens
    ADD COLUMN IF NOT EXISTS user_account_id UUID REFERENCES user_accounts(id);

ALTER TABLE auth_password_reset_tokens DROP COLUMN IF EXISTS auth_identity_id;
ALTER TABLE auth_email_verification_tokens DROP COLUMN IF EXISTS auth_identity_id;

DROP TABLE IF EXISTS auth_identities;

DROP INDEX IF EXISTS ix_user_accounts_primary_email;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'user_accounts'
          AND column_name = 'primary_email'
    ) THEN
        ALTER TABLE user_accounts RENAME COLUMN primary_email TO email;
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE user_accounts ADD COLUMN IF NOT EXISTS password_hash TEXT;
ALTER TABLE user_accounts ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ;
ALTER TABLE user_accounts ADD COLUMN IF NOT EXISTS password_changed_at TIMESTAMPTZ;

CREATE UNIQUE INDEX IF NOT EXISTS uq_user_accounts_email_active
    ON user_accounts (email)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_auth_email_verification_tokens_user_status
    ON auth_email_verification_tokens (user_account_id, status);

CREATE INDEX IF NOT EXISTS ix_auth_password_reset_tokens_user_status
    ON auth_password_reset_tokens (user_account_id, status);
