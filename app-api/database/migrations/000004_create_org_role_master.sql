-- +goose Up

CREATE TABLE org_roles (
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
    CONSTRAINT uq_org_roles_code   UNIQUE (code),
    CONSTRAINT ck_org_roles_code   CHECK (code ~ '^[a-z][a-z0-9_]{1,62}$'),
    CONSTRAINT ck_org_roles_status CHECK (status IN ('active', 'deprecated')),
    CONSTRAINT ck_org_roles_sort   CHECK (sort_order >= 0)
);

INSERT INTO org_roles (id, code, label_th, label_en, sort_order, is_system)
VALUES
    ('019e6d26-59ae-7992-9661-e90b5309fafc', 'owner',     'เจ้าของ',        'Owner',     10, true),
    ('019e6d26-59af-7c66-b98f-f4b363bac71c', 'admin',     'ผู้ดูแลระบบ',    'Admin',     20, true),
    ('019e6d26-59b1-792e-a978-3376f555a09b', 'executive', 'ผู้บริหาร',      'Executive', 30, true),
    ('019e6d26-59b2-7654-bfc1-131228f32b9a', 'user',      'ผู้ใช้งาน',      'User',      40, true);

-- +goose Down

DROP TABLE IF EXISTS org_roles;
