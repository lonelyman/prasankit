import type { Lang } from "@/lib/i18n";
import { t } from "@/lib/i18n";
import type { SelectOption } from "@/components/ui"; // REUSE — do not redeclare

// Order = sort_order from migration 000009; render in this order.
export const PROJECT_STATUS_CODES = [
  "draft",
  "planning",
  "proposal",
  "active",
  "closing",
  "maintenance",
  "closed",
  "archived",
] as const;
export const PROJECT_TYPE_CODES = ["internal", "client"] as const;
export type ProjectStatusCode = (typeof PROJECT_STATUS_CODES)[number];
export type ProjectTypeCode = (typeof PROJECT_TYPE_CODES)[number];

export function statusOptions(lang: Lang): SelectOption[] {
  return PROJECT_STATUS_CODES.map((c) => ({ value: c, label: t(`status.${c}`, lang) }));
}

export function typeOptions(lang: Lang): SelectOption[] {
  return PROJECT_TYPE_CODES.map((c) => ({ value: c, label: t(`type.${c}`, lang) }));
}

// FE-only display map: status code → Tailwind bg+text classes for the Badge.
// NOT a BE concept; purely a visual cue so the list scans at a glance.
const STATUS_COLOR: Record<ProjectStatusCode, string> = {
  draft: "bg-zinc-100 text-zinc-600",
  planning: "bg-indigo-100 text-indigo-700",
  proposal: "bg-violet-100 text-violet-700",
  active: "bg-green-100 text-green-700",
  closing: "bg-amber-100 text-amber-700",
  maintenance: "bg-sky-100 text-sky-700",
  closed: "bg-zinc-200 text-zinc-700",
  archived: "bg-zinc-100 text-zinc-400",
};

const STATUS_COLOR_FALLBACK = "bg-zinc-100 text-zinc-700";

export function statusColor(code: string): string {
  return STATUS_COLOR[code as ProjectStatusCode] ?? STATUS_COLOR_FALLBACK;
}

// Filter dropdowns prepend an "all" option {value:"", label:t("filter.all",lang)}.
export function statusFilterOptions(lang: Lang): SelectOption[] {
  return [{ value: "", label: t("filter.all", lang) }, ...statusOptions(lang)];
}

export function typeFilterOptions(lang: Lang): SelectOption[] {
  return [{ value: "", label: t("filter.all", lang) }, ...typeOptions(lang)];
}
