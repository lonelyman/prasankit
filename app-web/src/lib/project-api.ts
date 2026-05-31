import { apiGet, apiPost, apiPut, apiDelete } from "./api";
import type { Project, ProjectListPage } from "./types";

export interface ListProjectsParams {
  page?: number;
  limit?: number;
  project_status_code?: string;
  project_type_code?: string;
}

export interface CreateProjectInput {
  project_name: string;
  slug?: string; // slug OPTIONAL (BE stores NULL when blank)
  project_type_code: string;
  project_status_code: string;
  requesting_unit?: string;
  description?: string;
  start_date?: string;
  end_date?: string;
}

export interface UpdateProjectInput {
  // NO project_status_code — status changes go through changeProjectStatus.
  // PUT is a FULL REPLACE: omitting an optional field CLEARS it to NULL at the
  // BE (Updates(map[string]any{...}) writes every key incl. nil → SET NULL).
  // So clear-by-omit is correct; never send "" to clear (requesting_unit:"" → 422).
  project_name: string;
  slug?: string; // slug OPTIONAL
  project_type_code: string;
  requesting_unit?: string;
  description?: string;
  start_date?: string;
  end_date?: string;
}

const BASE = "/api/v1/workspaces/projects";

export async function listProjects(
  slug: string,
  params?: ListProjectsParams
): Promise<ProjectListPage> {
  const qs = new URLSearchParams();
  if (params?.page !== undefined) qs.append("page", String(params.page));
  if (params?.limit !== undefined) qs.append("limit", String(params.limit));
  if (params?.project_status_code) qs.append("project_status_code", params.project_status_code);
  if (params?.project_type_code) qs.append("project_type_code", params.project_type_code);
  const query = qs.toString();
  const path = query ? `${BASE}?${query}` : BASE;
  // apiGet already strips the outer .data → result is {items, pagination}.
  return apiGet<ProjectListPage>(path, { workspaceSlug: slug });
}

export async function createProject(slug: string, input: CreateProjectInput): Promise<Project> {
  const result = await apiPost<Project>(BASE, input, { workspaceSlug: slug });
  return result!;
}

export async function getProject(slug: string, id: string): Promise<Project> {
  return apiGet<Project>(`${BASE}/${id}`, { workspaceSlug: slug });
}

export async function updateProject(
  slug: string,
  id: string,
  input: UpdateProjectInput
): Promise<Project> {
  const result = await apiPut<Project>(`${BASE}/${id}`, input, { workspaceSlug: slug });
  return result!;
}

export async function deleteProject(slug: string, id: string): Promise<void> {
  await apiDelete(`${BASE}/${id}`, { workspaceSlug: slug });
}

export async function changeProjectStatus(
  slug: string,
  id: string,
  newStatusCode: string
): Promise<Project> {
  const result = await apiPost<Project>(
    `${BASE}/${id}/status`,
    { new_status_code: newStatusCode },
    { workspaceSlug: slug }
  );
  return result!;
}
