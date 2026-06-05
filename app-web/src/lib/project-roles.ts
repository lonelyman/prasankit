import type { Lang } from "@/lib/i18n";
import { t } from "@/lib/i18n";
import type { SelectOption } from "@/components/ui"; // REUSE — do not redeclare

// Assignable project roles ONLY (project_owner excluded — create-only, BE 422s it).
// Order = sort_order from migration 000009.
export const PROJECT_ROLE_CODES = ["project_manager", "member", "finance", "viewer"] as const;
export type ProjectRoleCode = (typeof PROJECT_ROLE_CODES)[number];

// Both selects (add + change-role) use these 4 codes — no project_owner.
export function projectRoleOptions(lang: Lang): SelectOption[] {
  return PROJECT_ROLE_CODES.map((c) => ({ value: c, label: t(`project_role.${c}`, lang) }));
}

// Handles project_owner too (the label exists) for rendering the owner row badge
// and the resolved owner block, even though it is excluded from the selects.
export function projectRoleLabel(code: string, lang: Lang): string {
  return t(`project_role.${code}`, lang);
}

// FE-only display map: project-role code → Tailwind bg+text classes for the Badge.
// NOT a BE concept; purely a visual cue. Mirrors statusColor in project-masters.ts.
const PROJECT_ROLE_COLOR: Record<string, string> = {
  project_owner: "bg-amber-100 text-amber-700",
  project_manager: "bg-indigo-100 text-indigo-700",
  member: "bg-zinc-100 text-zinc-700",
  finance: "bg-emerald-100 text-emerald-700",
  viewer: "bg-sky-100 text-sky-700",
};

const PROJECT_ROLE_COLOR_FALLBACK = "bg-zinc-100 text-zinc-700";

export function projectRoleColor(code: string): string {
  return PROJECT_ROLE_COLOR[code] ?? PROJECT_ROLE_COLOR_FALLBACK;
}
