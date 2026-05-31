-- +goose Up
-- M2 6b-2 (D39 junction + §2.11 FK-by-code). M:N project_member <-> project_position.
-- APPEND/DELETE-ONLY: no updated_at/updated_by/status/removed_at (history -> audit_logs).
-- DEPENDS ON 000012 (project_members backing UNIQUE) + 000014 (project_positions backing UNIQUE).
CREATE TABLE project_member_positions (
    id                    UUID        PRIMARY KEY,
    workspace_id          UUID        NOT NULL REFERENCES workspaces(id),
    project_member_id     UUID        NOT NULL,
    project_position_code TEXT        NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by            UUID        NOT NULL REFERENCES user_accounts(id),
    -- iron rule §M2.2.1: member + position must share workspace_id (DB-enforced).
    CONSTRAINT fk_pmp_project_member
        FOREIGN KEY (workspace_id, project_member_id)
        REFERENCES project_members (workspace_id, id),
    -- FK-by-code §2.11: position must be in the same workspace.
    CONSTRAINT fk_pmp_project_position
        FOREIGN KEY (workspace_id, project_position_code)
        REFERENCES project_positions (workspace_id, code)
);

-- prevent assigning the same position twice to one member.
CREATE UNIQUE INDEX uq_pmp_member_position
    ON project_member_positions (workspace_id, project_member_id, project_position_code);

-- reverse query: "who is `tech_lead` in this project/ws".
CREATE INDEX ix_pmp_position
    ON project_member_positions (workspace_id, project_position_code);

-- +goose Down
DROP TABLE IF EXISTS project_member_positions;
