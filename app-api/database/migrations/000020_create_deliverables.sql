-- +goose Up
-- M3 (§M3.2.2). deliverables = งวดงาน (ws-scoped business, soft-delete). title/due_date/sort_order.
-- composite FK (iron rule §M2.2.1) → projects(workspace_id, id) = DB-level isolation backstop.
-- ไม่มี status column (derive, D59) · ไม่มี assignee/reviewer column (M4).
CREATE TABLE deliverables (
    id           UUID         PRIMARY KEY,
    workspace_id UUID         NOT NULL REFERENCES workspaces(id),
    project_id   UUID         NOT NULL,
    title        TEXT         NOT NULL,
    description  TEXT         NULL,
    due_date     DATE         NULL,
    sort_order   INTEGER      NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by   UUID         NOT NULL REFERENCES user_accounts(id),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by   UUID         NULL REFERENCES user_accounts(id),
    deleted_at   TIMESTAMPTZ  NULL,
    deleted_by   UUID         NULL REFERENCES user_accounts(id),
    -- iron rule §M2.2.1: งวด + project ต้องอยู่ workspace เดียวกัน (DB-enforced).
    CONSTRAINT fk_deliverables_project
        FOREIGN KEY (workspace_id, project_id)
        REFERENCES projects (workspace_id, id),
    CONSTRAINT ck_deliverables_title_length
        CHECK (length(title) BETWEEN 1 AND 200),
    CONSTRAINT ck_deliverables_description_length
        CHECK (description IS NULL OR length(description) <= 10000),
    CONSTRAINT ck_deliverables_sort_order
        CHECK (sort_order >= 0)
);

-- Backing composite UNIQUE (non-partial): FK target for submissions.deliverable_id
-- composite (workspace_id, deliverable_id). Non-partial so a soft-deleted row is still
-- a valid FK target (service guards deleted_at, not the FK — precedent uq_projects/uq_members).
ALTER TABLE deliverables
    ADD CONSTRAINT uq_deliverables_workspace_id_id
    UNIQUE (workspace_id, id);

-- "งวดของ project นี้ เรียงด้วยมือ" (list order driver).
CREATE INDEX ix_deliverables_project_sort
    ON deliverables (workspace_id, project_id, sort_order)
    WHERE deleted_at IS NULL;

-- "งวดที่มีกำหนดส่ง" (due-date queries; M5 reminder/penalty hook).
CREATE INDEX ix_deliverables_project_due
    ON deliverables (workspace_id, project_id, due_date)
    WHERE deleted_at IS NULL AND due_date IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS deliverables;
