import { apiGet, apiPost, apiPut, apiDelete } from "./api";
import type { ProjectMember } from "./types";

export interface AddMemberInput {
  workspace_membership_id: string; // uuid
  project_role_code: string; // one of project_manager|member|finance|viewer (NOT project_owner)
}

export interface MemberListResult {
  items: ProjectMember[];
  count: number; // BE returns {items,count} — NON-paginated; no pagination envelope
}

const BASE = "/api/v1/workspaces/projects";

export async function listMembers(slug: string, projectId: string): Promise<MemberListResult> {
  // apiGet strips outer .data → {items, count}. display_name PRESENT on every item here.
  return apiGet<MemberListResult>(`${BASE}/${projectId}/members`, { workspaceSlug: slug });
}

export async function addMember(
  slug: string,
  projectId: string,
  input: AddMemberInput
): Promise<ProjectMember> {
  const result = await apiPost<ProjectMember>(`${BASE}/${projectId}/members`, input, {
    workspaceSlug: slug,
  });
  return result!; // 201 → data present (no display_name on this response — refetch list for the name)
}

export async function changeMemberRole(
  slug: string,
  projectId: string,
  memberId: string,
  newRoleCode: string
): Promise<ProjectMember> {
  const result = await apiPut<ProjectMember>(
    `${BASE}/${projectId}/members/${memberId}/role`,
    { new_role_code: newRoleCode },
    { workspaceSlug: slug }
  );
  return result!;
  // 200 (no display_name).
  // same-role on a NON-owner row → 200 no-op (no audit row, no UPDATE).
  // same-role on the OWNER row → 422 owner_immutable: the owner-immutable guard runs
  // BEFORE the no-op check in BE service.go ChangeRole. FE pre-disables the owner Select
  // so this should never fire; treat any owner_immutable as a stale-client reconcile path.
}

export async function removeMember(
  slug: string,
  projectId: string,
  memberId: string
): Promise<void> {
  await apiDelete(`${BASE}/${projectId}/members/${memberId}`, { workspaceSlug: slug });
  // 204 empty body — apiDelete handles; do NOT parse JSON.
}
