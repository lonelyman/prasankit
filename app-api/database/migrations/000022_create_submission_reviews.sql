-- +goose Up
-- M3 (§M3.2.4, D58/D60/D61). submission_reviews = ตรวจรับ (APPEND-ONLY, 1 review/submission).
-- APPEND-ONLY: ไม่มี updated_at/updated_by/deleted_at (verdict immutable; ผิด = ส่งรอบใหม่ + review ใหม่).
-- decision_code = FK-by-code → submission_decisions(code). reviewed_by = composite FK →
-- project_members (D60, ปิด M2.8 deliverable_reviewer FK); server-derive จาก actor (§5.4).
CREATE TABLE submission_reviews (
    id            UUID         PRIMARY KEY,
    workspace_id  UUID         NOT NULL REFERENCES workspaces(id),
    project_id    UUID         NOT NULL,
    submission_id UUID         NOT NULL,
    decision_code TEXT         NOT NULL REFERENCES submission_decisions(code),
    comment       TEXT         NULL,
    reviewed_by   UUID         NOT NULL,
    reviewed_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by    UUID         NOT NULL REFERENCES user_accounts(id),
    -- iron rule §M2.2.1: review/project/submission/ผู้ตรวจ ต้องอยู่ workspace เดียวกัน (DB-enforced).
    CONSTRAINT fk_submission_reviews_project
        FOREIGN KEY (workspace_id, project_id)
        REFERENCES projects (workspace_id, id),
    CONSTRAINT fk_submission_reviews_submission
        FOREIGN KEY (workspace_id, submission_id)
        REFERENCES submissions (workspace_id, id),
    CONSTRAINT fk_submission_reviews_reviewed_by
        FOREIGN KEY (workspace_id, reviewed_by)
        REFERENCES project_members (workspace_id, id),
    CONSTRAINT ck_submission_reviews_comment_length
        CHECK (comment IS NULL OR length(comment) <= 10000)
);

-- 1 review/submission (verdict immutable; กัน double-review race — backstop ใต้ FOR UPDATE D64).
ALTER TABLE submission_reviews
    ADD CONSTRAINT uq_submission_reviews_submission
    UNIQUE (workspace_id, submission_id);

-- NOTE: ไม่มี ix_submission_reviews_decision โดยตั้งใจ — decision rollup/KPI = M5 dashboard (§M3.9).

-- +goose Down
DROP TABLE IF EXISTS submission_reviews;
