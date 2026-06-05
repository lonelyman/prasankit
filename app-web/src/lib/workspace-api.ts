import { apiGet, apiPost } from "./api";
import type {
  WorkspaceWithRole,
  Workspace,
  CurrentWorkspace,
  Invitation,
  AcceptResult,
  WorkspaceMember,
} from "./types";

export async function listWorkspaces(): Promise<WorkspaceWithRole[]> {
  const result = await apiGet<{ items: WorkspaceWithRole[] }>("/api/v1/workspaces");
  return result.items;
}

export async function createWorkspace(input: {
  name: string;
  slug: string;
  contact_email: string;
}): Promise<Workspace> {
  const result = await apiPost<Workspace>("/api/v1/workspaces", input);
  // POST /workspaces returns 201 with data — result should not be null
  return result!;
}

export async function getCurrentWorkspace(slug: string): Promise<CurrentWorkspace> {
  return apiGet<CurrentWorkspace>("/api/v1/workspaces/current", {
    workspaceSlug: slug,
  });
}

export async function inviteMember(
  slug: string,
  input: { email: string; org_role_code: string }
): Promise<Invitation> {
  const result = await apiPost<Invitation>(
    "/api/v1/workspaces/invitations",
    input,
    { workspaceSlug: slug }
  );
  return result!;
}

// Add-member picker source: active workspace memberships with display_name.
// GET /api/v1/workspaces/members → {data:{items,count}}; apiGet strips .data.
// Gated owner/admin at the BE (same permission as invite).
export async function listWorkspaceMembers(
  slug: string
): Promise<{ items: WorkspaceMember[]; count: number }> {
  return apiGet<{ items: WorkspaceMember[]; count: number }>("/api/v1/workspaces/members", {
    workspaceSlug: slug,
  });
}

export async function acceptInvitation(token: string): Promise<AcceptResult> {
  const result = await apiPost<AcceptResult>("/api/v1/invitations/accept", {
    token,
  });
  return result!;
}
