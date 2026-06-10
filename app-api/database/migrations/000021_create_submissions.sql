-- +goose Up
-- M3 (§M3.2.3, D60/D61/D64). submissions = ส่งจริงหลายรอบ (ws-scoped business, APPEND-ONLY).
-- APPEND-ONLY: ไม่มี updated_at/updated_by/deleted_at (แก้ = ส่งรอบใหม่; precedent 000015).
-- round_no = service assign MAX+1 ใต้ deliverable FOR UPDATE (ไม่รับจาก client).
-- submitted_by = composite FK → project_members (D60); server-derive จาก actor (§5.4).
CREATE TABLE submissions (
    id              UUID         PRIMARY KEY,
    workspace_id    UUID         NOT NULL REFERENCES workspaces(id),
    project_id      UUID         NOT NULL,
    deliverable_id  UUID         NOT NULL,
    round_no        INTEGER      NOT NULL,
    note            TEXT         NULL,
    url             TEXT         NULL,
    submitted_by    UUID         NOT NULL,
    submitted_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by      UUID         NOT NULL REFERENCES user_accounts(id),
    -- iron rule §M2.2.1: งวด/project/ผู้ส่ง ต้องอยู่ workspace เดียวกัน (DB-enforced).
    CONSTRAINT fk_submissions_project
        FOREIGN KEY (workspace_id, project_id)
        REFERENCES projects (workspace_id, id),
    CONSTRAINT fk_submissions_deliverable
        FOREIGN KEY (workspace_id, deliverable_id)
        REFERENCES deliverables (workspace_id, id),
    CONSTRAINT fk_submissions_submitted_by
        FOREIGN KEY (workspace_id, submitted_by)
        REFERENCES project_members (workspace_id, id),
    CONSTRAINT ck_submissions_round_no
        CHECK (round_no >= 1),
    CONSTRAINT ck_submissions_note_length
        CHECK (note IS NULL OR length(note) <= 10000),
    CONSTRAINT ck_submissions_url_length
        CHECK (url IS NULL OR length(url) <= 2000)
);

-- round uniqueness / race backstop (D64): 1 row ต่อ round ต่องวด. fire = lock หาย = bug.
ALTER TABLE submissions
    ADD CONSTRAINT uq_submissions_deliverable_round
    UNIQUE (workspace_id, deliverable_id, round_no);

-- Backing composite UNIQUE (non-partial, คนละตัวกับ round): FK target for
-- submission_reviews.submission_id composite (workspace_id, submission_id).
ALTER TABLE submissions
    ADD CONSTRAINT uq_submissions_workspace_id_id
    UNIQUE (workspace_id, id);

-- list รอบของงวด (ล่าสุดบนสุด = derive driver / MAX(round_no)).
CREATE INDEX ix_submissions_deliverable
    ON submissions (workspace_id, deliverable_id, round_no DESC);

-- +goose Down
DROP TABLE IF EXISTS submissions;
