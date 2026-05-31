-- +goose Up
-- D40 (refined IMMEDIATE, NOT DEFERRABLE). Composite FK so the project owner is
-- a project_member of the SAME workspace. owner_project_member_id is nullable, so
-- the standard 3-step create sequence passes every statement under IMMEDIATE:
--   INSERT projects (owner=NULL)  -- NULL, no FK check
--   INSERT project_members        -- the owner row now exists
--   UPDATE projects SET owner_id  -- target exists -> passes
-- Target = project_members(workspace_id, id) via uq_project_members_workspace_id_id (000012).
-- DEPENDS ON 000012 (project_members + its backing UNIQUE must exist first).
ALTER TABLE projects
    ADD CONSTRAINT fk_projects_owner_project_member
    FOREIGN KEY (workspace_id, owner_project_member_id)
    REFERENCES project_members (workspace_id, id);

-- +goose Down
ALTER TABLE projects
    DROP CONSTRAINT IF EXISTS fk_projects_owner_project_member;
