import type { Lang } from "@/lib/i18n";
import type { Position } from "@/lib/types";

// FE-only display map: position status → Tailwind bg+text classes for the Badge.
// NOT a BE concept; purely a visual cue (deprecated rows read as muted).
const STATUS_COLOR: Record<string, string> = {
  active: "bg-green-100 text-green-700",
  deprecated: "bg-zinc-100 text-zinc-400",
};
const STATUS_COLOR_FALLBACK = "bg-zinc-100 text-zinc-700";

export function positionStatusColor(status: string): string {
  return STATUS_COLOR[status] ?? STATUS_COLOR_FALLBACK;
}

// Active-language label (positions carry both label_th and label_en inline, D27).
export function displayPositionLabel(p: Position, lang: Lang): string {
  return lang === "th" ? p.label_th : p.label_en;
}

// Order positions for display: sort_order ascending, then the EN label as a
// stable tiebreak. Pure — returns a new array, never mutates the input.
export function sortPositions(items: Position[]): Position[] {
  return [...items].sort((a, b) => {
    if (a.sort_order !== b.sort_order) return a.sort_order - b.sort_order;
    return a.label_en.localeCompare(b.label_en);
  });
}
