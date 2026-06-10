-- +goose Up
-- M3 (D58/§M3.2.1). submission_decisions = vocabulary ตรวจรับ (accept/conditional/reject).
-- Global system master (is_system=true, ไม่มี workspace_id). master shape §2.2 (เหมือน 000009).
-- FK target = submission_decisions(code) สำหรับ submission_reviews.decision_code (000022).
CREATE TABLE submission_decisions (
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
    CONSTRAINT uq_submission_decisions_code   UNIQUE (code),
    CONSTRAINT ck_submission_decisions_code   CHECK (code ~ '^[a-z][a-z0-9_]{1,62}$'),
    CONSTRAINT ck_submission_decisions_status CHECK (status IN ('active', 'deprecated')),
    CONSTRAINT ck_submission_decisions_sort   CHECK (sort_order >= 0)
);

INSERT INTO submission_decisions (id, code, label_th, label_en, sort_order, description, is_system, status)
VALUES
    ('019eb238-e3db-705d-ac4c-7e7f5305b6a8', 'accepted',    'ผ่าน',                'Accepted',    10, 'ตรวจรับผ่าน (terminal — ส่งซ้ำไม่ได้)', true, 'active'),
    ('019eb238-e3db-723e-a1bd-50c5de5ecd63', 'conditional', 'ผ่านแบบมีเงื่อนไข',  'Conditional', 20, 'ผ่านแบบมีเงื่อนไข (ส่งรอบใหม่ได้)',     true, 'active'),
    ('019eb238-e3db-7274-9c92-b88563b2afbc', 'rejected',    'ตีกลับ',              'Rejected',    30, 'ตีกลับให้แก้ไข (ส่งรอบใหม่ได้)',        true, 'active');

-- +goose Down
DROP TABLE IF EXISTS submission_decisions;
