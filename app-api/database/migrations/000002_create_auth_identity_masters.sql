-- +goose Up

CREATE TABLE auth_identity_types (
    id          UUID        PRIMARY KEY,
    code        TEXT        NOT NULL,
    label_th    TEXT        NOT NULL,
    label_en    TEXT        NOT NULL,
    description TEXT,
    sort_order  INT         NOT NULL DEFAULT 0,
    is_system   BOOLEAN     NOT NULL DEFAULT true,
    status      TEXT        NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_auth_identity_types_code   UNIQUE (code),
    CONSTRAINT ck_auth_identity_types_code   CHECK (code ~ '^[a-z][a-z0-9_]{1,62}$'),
    CONSTRAINT ck_auth_identity_types_status CHECK (status IN ('active', 'deprecated')),
    CONSTRAINT ck_auth_identity_types_sort   CHECK (sort_order >= 0)
);

INSERT INTO auth_identity_types (id, code, label_th, label_en, sort_order, is_system)
VALUES
    ('019e6d26-59a6-78bc-9e4d-c6a8ce8b88fa', 'email_password', 'อีเมล/รหัสผ่าน', 'Email & Password', 10, true);

CREATE TABLE account_statuses (
    id          UUID        PRIMARY KEY,
    code        TEXT        NOT NULL,
    label_th    TEXT        NOT NULL,
    label_en    TEXT        NOT NULL,
    description TEXT,
    sort_order  INT         NOT NULL DEFAULT 0,
    is_system   BOOLEAN     NOT NULL DEFAULT true,
    status      TEXT        NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_account_statuses_code   UNIQUE (code),
    CONSTRAINT ck_account_statuses_code   CHECK (code ~ '^[a-z][a-z0-9_]{1,62}$'),
    CONSTRAINT ck_account_statuses_status CHECK (status IN ('active', 'deprecated')),
    CONSTRAINT ck_account_statuses_sort   CHECK (sort_order >= 0)
);

INSERT INTO account_statuses (id, code, label_th, label_en, sort_order, is_system)
VALUES
    ('019e6d26-59a8-7d38-8584-4cba39253477', 'pending_verification', 'รอยืนยัน',       'Pending Verification', 10, true),
    ('019e6d26-59a9-7be2-ba41-1ca6dd96291e', 'active',               'ใช้งาน',          'Active',               20, true),
    ('019e6d26-59aa-7d34-8b8e-559db801073e', 'suspended',            'ระงับชั่วคราว',   'Suspended',            30, true),
    ('019e6d26-59ab-7fcd-9188-cf400b017c24', 'disabled',             'ปิดใช้งาน',       'Disabled',             40, true),
    ('019e6d26-59ad-72a3-8d8d-c99e1578eef4', 'deleted',              'ลบแล้ว',          'Deleted',              50, true);

-- +goose Down

DROP TABLE IF EXISTS account_statuses;
DROP TABLE IF EXISTS auth_identity_types;
