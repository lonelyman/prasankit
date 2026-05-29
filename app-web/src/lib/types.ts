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
