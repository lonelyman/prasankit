-- +goose Up
CREATE TABLE project_positions (
    id UUID PRIMARY KEY,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_system BOOLEAN NOT NULL DEFAULT true,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_project_positions_code UNIQUE (code),
    CONSTRAINT ck_project_positions_code_format CHECK (
        code ~ '^[a-z][a-z0-9_]{1,62}$'
    ),
    CONSTRAINT ck_project_positions_status CHECK (
        status IN ('active', 'deprecated')
    ),
    CONSTRAINT ck_project_positions_sort_order CHECK (sort_order >= 0)
);

INSERT INTO project_positions (id, code, name, description, sort_order, is_system)
VALUES
    ('019dd8a1-0640-7001-aee0-44d362881001', 'project_lead', 'Project Lead', 'Leads delivery execution', 10, true),
    ('019dd8a1-0640-7002-aee0-44d362881002', 'business_analyst', 'Business Analyst', 'Analyzes requirements and business process', 20, true),
    ('019dd8a1-0640-7003-aee0-44d362881003', 'developer', 'Developer', 'Builds and maintains implementation work', 30, true),
    ('019dd8a1-0640-7004-aee0-44d362881004', 'designer', 'Designer', 'Designs user experience or visual assets', 40, true),
    ('019dd8a1-0640-7005-aee0-44d362881005', 'tester', 'Tester', 'Tests quality and acceptance criteria', 50, true),
    ('019dd8a1-0640-7006-aee0-44d362881006', 'devops', 'DevOps', 'Handles deployment and runtime operations', 60, true),
    ('019dd8a1-0640-7007-aee0-44d362881007', 'finance_contact', 'Finance Contact', 'Coordinates budget and finance items', 70, true),
    ('019dd8a1-0640-7008-aee0-44d362881008', 'stakeholder', 'Stakeholder', 'Business or decision stakeholder', 80, true);

CREATE TABLE project_member_positions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    project_id UUID NOT NULL REFERENCES projects(id),
    project_member_id UUID NOT NULL REFERENCES project_members(id),
    position_id UUID NOT NULL REFERENCES project_positions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by UUID REFERENCES user_accounts(id),
    CONSTRAINT uq_project_member_positions_member_position UNIQUE (tenant_id, project_member_id, position_id)
);

CREATE INDEX ix_project_member_positions_member
    ON project_member_positions (tenant_id, workspace_id, project_id, project_member_id);

CREATE INDEX ix_project_member_positions_position
    ON project_member_positions (position_id);

-- +goose Down
DROP TABLE IF EXISTS project_member_positions;
DROP TABLE IF EXISTS project_positions;
