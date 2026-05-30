-- +goose Up

CREATE TABLE project_statuses (
    id          UUID        PRIMARY KEY,
    code        TEXT        NOT NULL,
    label_th    TEXT        NOT NULL,
    label_en    TEXT        NOT NULL,
    description TEXT,
    sort_order  INTEGER     NOT NULL DEFAULT 0,
    status      TEXT        NOT NULL DEFAULT 'active',
    is_system   BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_project_statuses_code   UNIQUE (code),
    CONSTRAINT ck_project_statuses_code   CHECK (code ~ '^[a-z][a-z0-9_]{1,62}$'),
    CONSTRAINT ck_project_statuses_status CHECK (status IN ('active', 'deprecated')),
    CONSTRAINT ck_project_statuses_sort   CHECK (sort_order >= 0)
);

INSERT INTO project_statuses (id, code, label_th, label_en, sort_order, description, is_system, status)
VALUES
    ('019e7904-8ae8-739d-b31c-da217d6b78ec', 'draft',       'ฉบับร่าง',         'Draft',       10, 'ระยะร่าง ยังไม่เริ่มดำเนินงาน',   true, 'active'),
    ('019e7904-8ae8-759d-b92c-dc741819542b', 'planning',    'วางแผน',           'Planning',    20, 'กำลังวางแผนโครงการ',              true, 'active'),
    ('019e7904-8ae8-75ff-9e1a-02846dbfbeb6', 'proposal',    'เสนอราคา',         'Proposal',    30, 'เสนอราคา/รออนุมัติ',              true, 'active'),
    ('019e7904-8ae8-7622-bc2a-4022fcab1179', 'active',      'กำลังดำเนินการ',   'Active',      40, 'กำลังทำงาน',                      true, 'active'),
    ('019e7904-8ae8-7641-907c-bbc1361d2498', 'closing',     'กำลังปิดงาน',      'Closing',     50, 'กำลังเก็บงานก่อนปิด',             true, 'active'),
    ('019e7904-8ae8-765c-bc05-17c33b48b806', 'maintenance', 'ดูแลรักษา',        'Maintenance', 60, 'อยู่ในช่วงดูแลหลังส่งมอบ',         true, 'active'),
    ('019e7904-8ae8-767c-9894-f91e5c84521e', 'closed',      'ปิดโครงการ',       'Closed',      70, 'ปิดเรียบร้อย',                    true, 'active'),
    ('019e7904-8ae8-7697-b8cb-7e7f3ae8154c', 'archived',    'เก็บถาวร',         'Archived',    80, 'เก็บถาวร ไม่แสดงใน default view', true, 'active');

CREATE TABLE project_types (
    id          UUID        PRIMARY KEY,
    code        TEXT        NOT NULL,
    label_th    TEXT        NOT NULL,
    label_en    TEXT        NOT NULL,
    description TEXT,
    sort_order  INTEGER     NOT NULL DEFAULT 0,
    status      TEXT        NOT NULL DEFAULT 'active',
    is_system   BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_project_types_code   UNIQUE (code),
    CONSTRAINT ck_project_types_code   CHECK (code ~ '^[a-z][a-z0-9_]{1,62}$'),
    CONSTRAINT ck_project_types_status CHECK (status IN ('active', 'deprecated')),
    CONSTRAINT ck_project_types_sort   CHECK (sort_order >= 0)
);

INSERT INTO project_types (id, code, label_th, label_en, sort_order, description, is_system, status)
VALUES
    ('019e7904-8ae8-76b2-8f85-ac363e749e0e', 'internal', 'ภายในองค์กร',     'Internal', 10, 'โครงการภายในของ workspace',         true, 'active'),
    ('019e7904-8ae8-76ce-b339-76f6f2ffe80d', 'client',   'งานลูกค้า',       'Client',   20, 'โครงการรับงานจากลูกค้าภายนอก',       true, 'active');

CREATE TABLE project_roles (
    id          UUID        PRIMARY KEY,
    code        TEXT        NOT NULL,
    label_th    TEXT        NOT NULL,
    label_en    TEXT        NOT NULL,
    description TEXT,
    sort_order  INTEGER     NOT NULL DEFAULT 0,
    status      TEXT        NOT NULL DEFAULT 'active',
    is_system   BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_project_roles_code   UNIQUE (code),
    CONSTRAINT ck_project_roles_code   CHECK (code ~ '^[a-z][a-z0-9_]{1,62}$'),
    CONSTRAINT ck_project_roles_status CHECK (status IN ('active', 'deprecated')),
    CONSTRAINT ck_project_roles_sort   CHECK (sort_order >= 0)
);

INSERT INTO project_roles (id, code, label_th, label_en, sort_order, is_system, status)
VALUES
    ('019e7904-8ae8-76e9-9099-46fb3f1152c6', 'project_owner',   'เจ้าของโครงการ',  'Project Owner',   10, true, 'active'),
    ('019e7904-8ae8-7704-9d21-87084d9aa333', 'project_manager', 'ผู้จัดการโครงการ', 'Project Manager', 20, true, 'active'),
    ('019e7904-8ae8-7720-9d28-b27c0c35b5f6', 'member',          'สมาชิก',          'Member',          30, true, 'active'),
    ('019e7904-8ae8-773b-bc70-5406330957df', 'finance',         'การเงิน',         'Finance',         40, true, 'active'),
    ('019e7904-8ae8-7756-8aad-3d8d44004e02', 'viewer',          'ผู้ดูข้อมูล',      'Viewer',          50, true, 'active');

-- +goose Down
DROP TABLE IF EXISTS project_roles;
DROP TABLE IF EXISTS project_types;
DROP TABLE IF EXISTS project_statuses;
