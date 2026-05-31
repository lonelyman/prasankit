-- +goose Up
-- M2 6b-2 (D41 + §2.11). 1:1 nullable company position per membership. composite FK so the
-- company_position is in the same ws as the membership. Existing M1 rows = NULL (no backfill);
-- composite FK is MATCH SIMPLE so a NULL component skips FK enforcement -> existing rows survive.
-- THIS IS MIGRATION 000016 (D45 renumber). NOTE: docs §M2.3.5 line 745 still labels this
-- "step 8 (000015)" -- that inline number is STALE post-D45 split; canonical = 000016 per §M2.6
-- step 9. Trust 000016; do NOT renumber to 000015 (that is the junction).
-- DEPENDS ON 000014 (company_positions backing UNIQUE) + 000008 (workspace_memberships backing UNIQUE).
-- Ops note: non-CONCURRENT index acceptable at M1/M2 scale; production would use CONCURRENTLY
-- outside the goose tx. Mirrors 000008.
ALTER TABLE workspace_memberships
    ADD COLUMN company_position_code TEXT;

ALTER TABLE workspace_memberships
    ADD CONSTRAINT fk_workspace_memberships_company_position
    FOREIGN KEY (workspace_id, company_position_code)
    REFERENCES company_positions (workspace_id, code);

CREATE INDEX ix_workspace_memberships_company_position
    ON workspace_memberships (workspace_id, company_position_code)
    WHERE company_position_code IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS ix_workspace_memberships_company_position;
ALTER TABLE workspace_memberships
    DROP CONSTRAINT IF EXISTS fk_workspace_memberships_company_position;
ALTER TABLE workspace_memberships
    DROP COLUMN IF EXISTS company_position_code;
