-- +goose Up
CREATE TABLE projects (
    id                       UUID         PRIMARY KEY,
    workspace_id             UUID         NOT NULL REFERENCES workspaces(id),
    project_name             TEXT         NOT NULL,
    slug                     CITEXT       NULL,
    project_type_code        TEXT         NOT NULL REFERENCES project_types(code),
    project_status_code      TEXT         NOT NULL REFERENCES project_statuses(code),
    -- owner_project_member_id: column declared nullable in 6a, NO FK yet.
    -- 6b's 000012_alter_projects_add_owner_fk.sql adds the composite FK
    -- (workspace_id, owner_project_member_id) -> project_members(workspace_id, id).
    -- In 6a this column MUST always be NULL on INSERT / UPDATE.
    owner_project_member_id  UUID         NULL,
    requesting_unit          TEXT         NULL,
    description              TEXT         NULL,
    start_date               DATE         NULL,
    end_date                 DATE         NULL,
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by               UUID         NOT NULL REFERENCES user_accounts(id),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by               UUID         NULL REFERENCES user_accounts(id),
    deleted_at               TIMESTAMPTZ  NULL,
    deleted_by               UUID         NULL REFERENCES user_accounts(id),
    CONSTRAINT ck_projects_project_name_length
        CHECK (length(project_name) BETWEEN 1 AND 200),
    CONSTRAINT ck_projects_slug_format
        CHECK (slug IS NULL OR slug::text ~ '^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$'),
    CONSTRAINT ck_projects_requesting_unit_length
        CHECK (requesting_unit IS NULL OR length(requesting_unit) BETWEEN 1 AND 200),
    CONSTRAINT ck_projects_description_length
        CHECK (description IS NULL OR length(description) <= 10000),
    CONSTRAINT ck_projects_date_range
        CHECK (start_date IS NULL OR end_date IS NULL OR start_date <= end_date)
);

CREATE UNIQUE INDEX uq_projects_workspace_slug_active
    ON projects (workspace_id, slug)
    WHERE deleted_at IS NULL AND slug IS NOT NULL;

CREATE INDEX ix_projects_workspace_status_created
    ON projects (workspace_id, project_status_code, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX ix_projects_workspace_type
    ON projects (workspace_id, project_type_code)
    WHERE deleted_at IS NULL;

-- ix_projects_workspace_owner: pre-created so the projects table schema is
-- stable across slices. Empty in 6a (owner_project_member_id is always NULL);
-- populated in 6b after owner_project_member_id is set.
CREATE INDEX ix_projects_workspace_owner
    ON projects (workspace_id, owner_project_member_id)
    WHERE deleted_at IS NULL;

-- Composite UNIQUE: backing target for 6b's project_members.project_id FK
-- composite (workspace_id, project_id) and for owner FK composite.
ALTER TABLE projects
    ADD CONSTRAINT uq_projects_workspace_id_id
    UNIQUE (workspace_id, id);

-- +goose Down
DROP TABLE IF EXISTS projects;
