-- +goose Up
-- Fix: the slug format regex `[a-z0-9-]{1,61}` accidentally forbade EXACTLY-2-char
-- slugs — the pattern matched 1 char OR 3-63 chars, never 2 (e.g. "ai" was rejected
-- with a misleading "format" error). Relax {1,61} -> {0,61} so 2-char slugs pass;
-- start/end must still be alphanumeric, hyphens only in the middle, max length 63.
-- Mirrors the FE (validation.ts SLUG_RE) + BE (workspace/project slugPattern) fix.
ALTER TABLE workspaces DROP CONSTRAINT IF EXISTS ck_workspaces_slug;
ALTER TABLE workspaces ADD CONSTRAINT ck_workspaces_slug
    CHECK (slug::text ~ '^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$');

ALTER TABLE projects DROP CONSTRAINT IF EXISTS ck_projects_slug_format;
ALTER TABLE projects ADD CONSTRAINT ck_projects_slug_format
    CHECK (slug IS NULL OR slug::text ~ '^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$');

-- +goose Down
ALTER TABLE projects DROP CONSTRAINT IF EXISTS ck_projects_slug_format;
ALTER TABLE projects ADD CONSTRAINT ck_projects_slug_format
    CHECK (slug IS NULL OR slug::text ~ '^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$');

ALTER TABLE workspaces DROP CONSTRAINT IF EXISTS ck_workspaces_slug;
ALTER TABLE workspaces ADD CONSTRAINT ck_workspaces_slug
    CHECK (slug::text ~ '^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$');
