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

// Filter dropdowns prepend an "all" option {value:"", label:t("filter.all",lang)}.
export function statusFilterOptions(lang: Lang): SelectOption[] {
  return [{ value: "", label: t("filter.all", lang) }, ...statusOptions(lang)];
}

export function typeFilterOptions(lang: Lang): SelectOption[] {
  return [{ value: "", label: t("filter.all", lang) }, ...typeOptions(lang)];
}
