-- +goose Up
-- Compatibility migration for databases that applied 000002 before the
-- primary key strategy changed to application-generated UUID v7.
ALTER TABLE user_accounts ALTER COLUMN id DROP DEFAULT;
ALTER TABLE auth_sessions ALTER COLUMN id DROP DEFAULT;
ALTER TABLE auth_login_attempts ALTER COLUMN id DROP DEFAULT;
ALTER TABLE auth_email_verification_tokens ALTER COLUMN id DROP DEFAULT;
ALTER TABLE auth_password_reset_tokens ALTER COLUMN id DROP DEFAULT;
ALTER TABLE security_events ALTER COLUMN id DROP DEFAULT;

-- +goose Down
ALTER TABLE user_accounts ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE auth_sessions ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE auth_login_attempts ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE auth_email_verification_tokens ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE auth_password_reset_tokens ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE security_events ALTER COLUMN id SET DEFAULT gen_random_uuid();
