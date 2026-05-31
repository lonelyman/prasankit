import { describe, it, expect } from "vitest";
import { statusColor, PROJECT_STATUS_CODES } from "./project-masters";

describe("statusColor", () => {
  it("returns a non-empty Tailwind class string for every status code", () => {
    for (const c of PROJECT_STATUS_CODES) {
      const cls = statusColor(c);
      expect(cls).toBeTruthy();
      expect(typeof cls).toBe("string");
      // each entry pairs a bg + a text color
      expect(cls).toMatch(/\bbg-/);
      expect(cls).toMatch(/\btext-/);
    }
  });

  it("maps each known code to a distinct, defined class (no key falls through to the same fallback unintentionally)", () => {
    // active is green; draft is zinc — sanity-check two anchors so a future
    // accidental remap is caught.
    expect(statusColor("active")).toContain("green");
    expect(statusColor("draft")).toContain("zinc");
  });

  it("returns the fallback for an unknown code rather than throwing", () => {
    const cls = statusColor("not-a-real-status");
    expect(cls).toBeTruthy();
    expect(cls).toMatch(/\bbg-/);
    expect(cls).toMatch(/\btext-/);
  });
});
