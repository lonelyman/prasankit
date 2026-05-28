-- +goose Up

CREATE TABLE workspace_statuses (
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
    CONSTRAINT uq_workspace_statuses_code   UNIQUE (code),
    CONSTRAINT ck_workspace_statuses_code   CHECK (code ~ '^[a-z][a-z0-9_]{1,62}$'),
    CONSTRAINT ck_workspace_statuses_status CHECK (status IN ('active', 'deprecated')),
    CONSTRAINT ck_workspace_statuses_sort   CHECK (sort_order >= 0)
);

INSERT INTO workspace_statuses (id, code, label_th, label_en, sort_order, is_system)
VALUES
    ('019e6d26-59b3-7f3d-b4ee-d2e72b54c293', 'active',           'ใช้งาน',   'Active',          10, true),
    ('019e6d26-59b4-7970-abfb-1ae653fe7a18', 'suspended',        'ระงับ',    'Suspended',       20, true),
    ('019e6d26-59b6-76b0-aa01-a8ed81a3bc4c', 'pending_deletion', 'รอลบ',     'Pending Deletion', 30, true),
    ('019e6d26-59b7-76e7-8b5e-f7808fa35315', 'deleted',          'ลบแล้ว',   'Deleted',         40, true);

CREATE TABLE membership_statuses (
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
    CONSTRAINT uq_membership_statuses_code   UNIQUE (code),
    CONSTRAINT ck_membership_statuses_code   CHECK (code ~ '^[a-z][a-z0-9_]{1,62}$'),
    CONSTRAINT ck_membership_statuses_status CHECK (status IN ('active', 'deprecated')),
    CONSTRAINT ck_membership_statuses_sort   CHECK (sort_order >= 0)
);

INSERT INTO membership_statuses (id, code, label_th, label_en, sort_order, is_system)
VALUES
    ('019e6d26-59b8-77ca-bf1f-1f43e02b7225', 'active',    'ใช้งาน',    'Active',    10, true),
    ('019e6d26-59ba-7669-ad18-f6630ee46da5', 'suspended', 'ระงับ',     'Suspended', 20, true),
    ('019e6d26-59bb-7e69-af6a-6f3f45fa8aea', 'removed',   'ถูกนำออก',  'Removed',   30, true);

-- +goose Down

DROP TABLE IF EXISTS membership_statuses;
DROP TABLE IF EXISTS workspace_statuses;
