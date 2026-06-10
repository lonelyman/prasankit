-- +goose Up
-- M3 (D59/§M3.2.1). deliverable_statuses = label dictionary ของ "สถานะงวด" ที่ DERIVE
-- (not_submitted/in_review/accepted/conditional/rejected). master shape §2.2 (เหมือน 000009).
--
-- ⚠️ DESIGN NOTE (D59): นี่คือ label dictionary ของ derived status — ไม่มี FK target
-- โดยตั้งใจ (ไม่มี table ใดชี้มา code นี้). status ของงวด = derive จาก (รอบล่าสุด, review
-- ของมัน) ที่ service/presenter ไม่ materialize เป็น column. ห้ามเพิ่ม
-- deliverable_status_code column บน deliverables โดยไม่ revisit D59 (กัน wrong-fix
-- reflex แบบ D47/D48) — materialize = profiling-gated optimization เท่านั้น.
CREATE TABLE deliverable_statuses (
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
    CONSTRAINT uq_deliverable_statuses_code   UNIQUE (code),
    CONSTRAINT ck_deliverable_statuses_code   CHECK (code ~ '^[a-z][a-z0-9_]{1,62}$'),
    CONSTRAINT ck_deliverable_statuses_status CHECK (status IN ('active', 'deprecated')),
    CONSTRAINT ck_deliverable_statuses_sort   CHECK (sort_order >= 0)
);

INSERT INTO deliverable_statuses (id, code, label_th, label_en, sort_order, description, is_system, status)
VALUES
    ('019eb238-e3db-7298-81e9-6735169c119b', 'not_submitted', 'ยังไม่ส่ง',          'Not Submitted', 10, 'ยังไม่มี submission',                  true, 'active'),
    ('019eb238-e3db-72bb-ac63-3891bd9af0c0', 'in_review',     'รอตรวจรับ',          'In Review',     20, 'มี submission รอบล่าสุด แต่ยังไม่ตรวจรับ', true, 'active'),
    ('019eb238-e3db-72da-9881-3962824b3837', 'accepted',      'ผ่าน',               'Accepted',      30, 'ตรวจรับรอบล่าสุด = ผ่าน',              true, 'active'),
    ('019eb238-e3db-72f5-8123-8105e228d385', 'conditional',   'ผ่านแบบมีเงื่อนไข', 'Conditional',   40, 'ตรวจรับรอบล่าสุด = ผ่านแบบมีเงื่อนไข', true, 'active'),
    ('019eb238-e3db-730d-8344-d2a86b7d30ba', 'rejected',      'ตีกลับ',             'Rejected',      50, 'ตรวจรับรอบล่าสุด = ตีกลับ',            true, 'active');

-- +goose Down
DROP TABLE IF EXISTS deliverable_statuses;
