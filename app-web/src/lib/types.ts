export interface Account {
  id: string;
  primary_email: string;
  display_name: string;
  account_status_code: string;
}

export interface Workspace {
  id: string;
  workspace_name: string;
  slug: string;
  workspace_status_code: string;
  contact_email: string;
}

export interface WorkspaceWithRole extends Workspace {
  org_role_code: string;
}

export interface CurrentWorkspace {
  id: string;
  workspace_name: string;
  slug: string;
  workspace_status_code: string;
  org_role_code: string;
}

export interface Invitation {
  id: string;
  workspace_id: string;
  email: string;
  org_role_code: string;
  expires_at: string;
}

export interface AcceptResult {
  workspace_id: string;
  org_role_code: string;
}

export interface Project {
  id: string;
  workspace_id: string;
  project_name: string;
  slug: string | null; // *string in BE; null when blank
  project_type_code: string;
  project_status_code: string;
  owner_project_member_id?: string | null; // omitempty: ABSENT when unset; non-null for projects created by this BE
  requesting_unit: string | null;
  description: string | null;
  start_date: string | null; // "YYYY-MM-DD"
  end_date: string | null; // "YYYY-MM-DD"
  created_at: string; // RFC3339
  updated_at: string; // RFC3339
}

export interface ProjectMember {
  id: string;
  workspace_id: string;
  project_id: string;
  workspace_membership_id: string;
  project_role_code: string;
  joined_at: string; // RFC3339
  removed_at: string | null; // key ALWAYS present (null when active); list returns active-only
  display_name?: string; // PRESENT on list items; ABSENT on add/change responses (omitempty)
}

// Add-member picker source. GET /api/v1/workspaces/members returns active
// memberships only with display_name; BE omits membership_status_code (already
// active-filtered), so no status field here — matches the live contract.
export interface WorkspaceMember {
  workspace_membership_id: string;
  display_name: string;
  org_role_code: string;
}

export interface ProjectPagination {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
  has_more: boolean;
}

export interface ProjectListPage {
  items: Project[];
  pagination: ProjectPagination;
}

// Workspace-scoped customizable master (project_positions / company_positions).
// Same shape for both lists; the kind only selects the endpoint. Masters start
// EMPTY (no seed) and are managed via /settings/positions — never hardcode codes.
export interface Position {
  id: string;
  workspace_id: string;
  code: string; // immutable, ^[a-z][a-z0-9_]{1,62}$
  label_th: string;
  label_en: string;
  description: string | null; // *string in BE; null when unset
  sort_order: number;
  is_system: boolean;
  status: string; // "active" | "deprecated"
  created_at: string; // RFC3339
  updated_at: string;
}

export interface PositionListPage {
  items: Position[];
  pagination: ProjectPagination; // reuse the shared pagination envelope
}
