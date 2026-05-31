-- +goose Up
-- M2 6b-1 (D38 iron rule + D40 owner target). project_members links an active
-- workspace_membership to a project with a project_role. removed_at = timestamp-
-- derived state (no status master, §2.6). Composite FKs are the DB-level
-- isolation backstop (invariant #4).
CREATE TABLE project_members (
    id                       UUID         PRIMARY KEY,
    workspace_id             UUID         NOT NULL REFERENCES workspaces(id),
    project_id               UUID         NOT NULL,
    workspace_membership_id  UUID         NOT NULL,
    project_role_code        TEXT         NOT NULL REFERENCES project_roles(code),
    joined_at                TIMESTAMPTZ  NOT NULL DEFAULT now(),
    removed_at               TIMESTAMPTZ  NULL,
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by               UUID         NOT NULL REFERENCES user_accounts(id),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by               UUID         NULL REFERENCES user_accounts(id),
    -- iron rule §M2.2.1: member + project must share workspace_id (DB-enforced).
    CONSTRAINT fk_project_members_project
        FOREIGN KEY (workspace_id, project_id)
        REFERENCES projects (workspace_id, id),
    CONSTRAINT fk_project_members_membership
        FOREIGN KEY (workspace_id, workspace_membership_id)
        REFERENCES workspace_memberships (workspace_id, id)
);

-- 1 membership = 1 ACTIVE row per project (re-add after remove = new row).
CREATE UNIQUE INDEX uq_project_members_active
    ON project_members (workspace_id, project_id, workspace_membership_id)
    WHERE removed_at IS NULL;

-- "team of this project" + role filter.
CREATE INDEX ix_project_members_project_role
    ON project_members (workspace_id, project_id, project_role_code)
    WHERE removed_at IS NULL;

-- "which projects is this member in (this ws)".
CREATE INDEX ix_project_members_membership
    ON project_members (workspace_id, workspace_membership_id)
    WHERE removed_at IS NULL;

-- Backing composite UNIQUE (non-partial): FK target for projects.owner_project_member_id
-- (000013) and for 6b-2 project_member_positions. Non-partial so a removed row is still
-- a valid FK target (service guards removed_at, not the FK — §M2.3.4).
ALTER TABLE project_members
    ADD CONSTRAINT uq_project_members_workspace_id_id
    UNIQUE (workspace_id, id);

-- +goose Down
DROP TABLE IF EXISTS project_members;
