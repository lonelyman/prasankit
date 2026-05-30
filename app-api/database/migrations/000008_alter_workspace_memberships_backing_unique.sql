-- +goose Up
-- D38 backing UNIQUE prerequisite for 6b composite FKs from project_members.
-- No-op on data (PK already guarantees `id` uniqueness); Postgres requires
-- explicit composite UNIQUE as FK target.
ALTER TABLE workspace_memberships
    ADD CONSTRAINT uq_workspace_memberships_workspace_id_id
    UNIQUE (workspace_id, id);

-- +goose Down
ALTER TABLE workspace_memberships
    DROP CONSTRAINT IF EXISTS uq_workspace_memberships_workspace_id_id;
