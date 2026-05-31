-- +goose Up
-- M2 6b-2 (D39 + §2.9 + §2.11). Two workspace-scoped CUSTOMIZABLE masters: project_positions
-- (M:N via project_member_positions) and company_positions (1:1 via workspace_memberships).
-- NOT seeded in M2 (vision §3 does not lock industry). code immutable + no-reuse-after-deprecate
-- enforced at SERVICE layer; DB holds: non-partial UNIQUE (FK target backing) + format CHECK +
-- status CHECK. status='active'|'deprecated' REPLACES soft-delete (§2.9). is_system=false default
-- (no seed); is_system guard is service-layer (M3+ may seed is_system=true rows).
-- Ops note: non-CONCURRENT index/constraint here is acceptable at M1/M2 ws-sized scale; a
-- production-scale table would use CREATE INDEX CONCURRENTLY + ADD CONSTRAINT ... USING INDEX
-- outside the goose tx (a no-transaction migration). Mirrors the 000008 ops note. No SQL change for M2.
CREATE TABLE project_positions (
    id          UUID        PRIMARY KEY,
    workspace_id UUID       NOT NULL REFERENCES workspaces(id),
    code        TEXT        NOT NULL,
    label_th    TEXT        NOT NULL,
    label_en    TEXT        NOT NULL,
    description TEXT,
    sort_order  INTEGER     NOT NULL DEFAULT 0,
    is_system   BOOLEAN     NOT NULL DEFAULT false,
    status      TEXT        NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by  UUID        NOT NULL REFERENCES user_accounts(id),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by  UUID        NULL REFERENCES user_accounts(id),
    CONSTRAINT ck_project_positions_code   CHECK (code ~ '^[a-z][a-z0-9_]{1,62}$'),
    CONSTRAINT ck_project_positions_status CHECK (status IN ('active', 'deprecated')),
    CONSTRAINT ck_project_positions_sort   CHECK (sort_order >= 0)
);

-- Backing UNIQUE for the composite FK target project_member_positions.(workspace_id, project_position_code)
-- (§2.11). Use ADD CONSTRAINT ... UNIQUE (NOT a bare CREATE UNIQUE INDEX) to MATCH the committed
-- backing-unique convention (000008/000010/000012 all use ADD CONSTRAINT ... UNIQUE). The constraint's
-- implicit backing index also serves per-ws (workspace_id, code) uniqueness + lookups; the separate
-- status index below serves the picker filter. Non-partial: covers active+deprecated (no code reuse).
-- (NB: a non-partial unique INDEX is *also* a valid composite-FK target in Postgres — empirically
--  confirmed PG16; only PARTIAL unique indexes are rejected as FK targets — so the index form would
--  NOT have broken; the constraint form is chosen purely for codebase convention + tooling clarity.)
ALTER TABLE project_positions
    ADD CONSTRAINT uq_project_positions_workspace_code UNIQUE (workspace_id, code);

-- list picker filter status='active'.
CREATE INDEX ix_project_positions_workspace_status
    ON project_positions (workspace_id, status);

CREATE TABLE company_positions (
    id          UUID        PRIMARY KEY,
    workspace_id UUID       NOT NULL REFERENCES workspaces(id),
    code        TEXT        NOT NULL,
    label_th    TEXT        NOT NULL,
    label_en    TEXT        NOT NULL,
    description TEXT,
    sort_order  INTEGER     NOT NULL DEFAULT 0,
    is_system   BOOLEAN     NOT NULL DEFAULT false,
    status      TEXT        NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by  UUID        NOT NULL REFERENCES user_accounts(id),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by  UUID        NULL REFERENCES user_accounts(id),
    CONSTRAINT ck_company_positions_code   CHECK (code ~ '^[a-z][a-z0-9_]{1,62}$'),
    CONSTRAINT ck_company_positions_status CHECK (status IN ('active', 'deprecated')),
    CONSTRAINT ck_company_positions_sort   CHECK (sort_order >= 0)
);

-- Backing UNIQUE for workspace_memberships.(workspace_id, company_position_code) (§2.11).
-- Same ADD CONSTRAINT ... UNIQUE convention + rationale as project_positions above.
ALTER TABLE company_positions
    ADD CONSTRAINT uq_company_positions_workspace_code UNIQUE (workspace_id, code);

CREATE INDEX ix_company_positions_workspace_status
    ON company_positions (workspace_id, status);

-- +goose Down
DROP TABLE IF EXISTS company_positions;
DROP TABLE IF EXISTS project_positions;
