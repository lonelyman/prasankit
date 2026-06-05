import { apiGet, apiPost, apiPut } from "./api";
import type { Position, PositionListPage } from "./types";

// The two workspace-scoped master lists share one client; "kind" picks the URL
// segment. project_positions = hats a member wears in a project (M:N);
// company_positions = a member's company-wide job title (1:1). Masters are
// customizable and start EMPTY (no seed) — codes/labels are always fetched live.
export type PositionKind = "project" | "company";

const BASES: Record<PositionKind, string> = {
  project: "/api/v1/workspaces/project-positions",
  company: "/api/v1/workspaces/company-positions",
};

export interface ListPositionsParams {
  status?: string; // "active" | "deprecated"; server-side filter, omit = all
  page?: number;
  limit?: number; // BE default 10, max 100
}

export interface CreatePositionInput {
  code: string;
  label_th: string;
  label_en: string;
  description?: string; // omitted from the body when blank (never sent as "")
  sort_order: number;
}

export interface UpdatePositionInput {
  code: string; // must equal the path code or be omitted (else 422 code_immutable)
  label_th: string;
  label_en: string;
  description?: string;
  sort_order: number;
  status: string; // "active" | "deprecated"
}

export async function listPositions(
  slug: string,
  kind: PositionKind,
  params?: ListPositionsParams
): Promise<PositionListPage> {
  const qs = new URLSearchParams();
  if (params?.status) qs.append("status", params.status);
  if (params?.page !== undefined) qs.append("page", String(params.page));
  if (params?.limit !== undefined) qs.append("limit", String(params.limit));
  const query = qs.toString();
  const path = query ? `${BASES[kind]}?${query}` : BASES[kind];
  // apiGet strips the outer .data → {items, pagination}.
  return apiGet<PositionListPage>(path, { workspaceSlug: slug });
}

export async function createPosition(
  slug: string,
  kind: PositionKind,
  input: CreatePositionInput
): Promise<Position> {
  const result = await apiPost<Position>(BASES[kind], toBody(input), { workspaceSlug: slug });
  return result!; // 201 → data present
}

export async function updatePosition(
  slug: string,
  kind: PositionKind,
  code: string,
  input: UpdatePositionInput
): Promise<Position> {
  const result = await apiPut<Position>(`${BASES[kind]}/${code}`, toBody(input), {
    workspaceSlug: slug,
  });
  return result!;
}

export async function deprecatePosition(
  slug: string,
  kind: PositionKind,
  code: string
): Promise<Position> {
  const result = await apiPost<Position>(`${BASES[kind]}/${code}/deprecate`, {}, {
    workspaceSlug: slug,
  });
  return result!;
}

// Build the request body, omitting an empty description so the BE stores NULL
// (never send "" — that would persist an empty string). On update, omitting a
// now-blank description clears it to NULL (full-replace semantics).
function toBody(input: CreatePositionInput | UpdatePositionInput): Record<string, unknown> {
  const body: Record<string, unknown> = {
    code: input.code,
    label_th: input.label_th,
    label_en: input.label_en,
    sort_order: input.sort_order,
  };
  if ("status" in input) body.status = input.status;
  const desc = input.description?.trim();
  if (desc) body.description = desc;
  return body;
}
