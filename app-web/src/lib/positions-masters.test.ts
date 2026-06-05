import { describe, it, expect } from "vitest";
import { positionStatusColor, displayPositionLabel, sortPositions } from "./positions-masters";
import type { Position } from "./types";

function pos(overrides: Partial<Position> = {}): Position {
  return {
    id: "x",
    workspace_id: "w",
    code: "c",
    label_th: "ไทย",
    label_en: "English",
    description: null,
    sort_order: 0,
    is_system: false,
    status: "active",
    created_at: "",
    updated_at: "",
    ...overrides,
  };
}

describe("positionStatusColor", () => {
  it("returns distinct non-empty classes for active and deprecated", () => {
    expect(positionStatusColor("active")).toBeTruthy();
    expect(positionStatusColor("deprecated")).toBeTruthy();
    expect(positionStatusColor("active")).not.toBe(positionStatusColor("deprecated"));
  });

  it("falls back to a non-empty class for an unknown status", () => {
    expect(positionStatusColor("weird")).toBeTruthy();
  });
});

describe("displayPositionLabel", () => {
  it("returns label_th for th and label_en for en", () => {
    const p = pos({ label_th: "หัวหน้า", label_en: "Lead" });
    expect(displayPositionLabel(p, "th")).toBe("หัวหน้า");
    expect(displayPositionLabel(p, "en")).toBe("Lead");
  });
});

describe("sortPositions", () => {
  it("orders by sort_order then label_en, without mutating the input", () => {
    const a = pos({ id: "a", sort_order: 2, label_en: "Alpha" });
    const b = pos({ id: "b", sort_order: 1, label_en: "Beta" });
    const c = pos({ id: "c", sort_order: 1, label_en: "Alpha" });
    const input = [a, b, c];

    const out = sortPositions(input);

    expect(out.map((p) => p.id)).toEqual(["c", "b", "a"]);
    expect(input.map((p) => p.id)).toEqual(["a", "b", "c"]); // input untouched
  });
});
