import { describe, it, expect } from "vitest";
import {
  validateEmail,
  validatePassword,
  validateConfirmPassword,
  validateDisplayName,
  validateNewPassword,
  validateWorkspaceName,
  validateSlug,
  validateProjectName,
  validateProjectSlugOptional,
  validateRequestingUnit,
  validateDescription,
  validateProjectDate,
  validateDateRange,
} from "./validation";

describe("validateEmail", () => {
  it("returns null for a valid email", () => {
    expect(validateEmail("user@example.com")).toBeNull();
  });

  it("returns null for a valid email with subdomains", () => {
    expect(validateEmail("user@mail.example.co.th")).toBeNull();
  });

  it("returns error key for missing @", () => {
    expect(validateEmail("notanemail")).toBe("invalid_email");
  });

  it("returns error key for empty string", () => {
    expect(validateEmail("")).toBe("invalid_email");
  });

  it("returns error key for whitespace only", () => {
    expect(validateEmail("   ")).toBe("invalid_email");
  });

  it("returns error key for email without domain", () => {
    expect(validateEmail("user@")).toBe("invalid_email");
  });
});

describe("validatePassword", () => {
  it("returns null for exactly 8 characters", () => {
    expect(validatePassword("12345678")).toBeNull();
  });

  it("returns null for more than 8 characters", () => {
    expect(validatePassword("averylongpassword")).toBeNull();
  });

  it("returns error key for 7 characters", () => {
    expect(validatePassword("1234567")).toBe("password_too_short");
  });

  it("returns error key for empty string", () => {
    expect(validatePassword("")).toBe("password_too_short");
  });
});

describe("validateConfirmPassword", () => {
  it("returns null when confirm matches password", () => {
    expect(validateConfirmPassword("secret123", "secret123")).toBeNull();
  });

  it("returns password_mismatch when they differ", () => {
    expect(validateConfirmPassword("secret123", "secret124")).toBe("password_mismatch");
  });

  it("returns password_mismatch when confirm is empty but password is not", () => {
    expect(validateConfirmPassword("secret123", "")).toBe("password_mismatch");
  });
});

describe("validateDisplayName", () => {
  it("returns null for a regular name", () => {
    expect(validateDisplayName("Alice")).toBeNull();
  });

  it("returns null for exactly 100 characters", () => {
    expect(validateDisplayName("a".repeat(100))).toBeNull();
  });

  it("returns error key for 101 characters", () => {
    expect(validateDisplayName("a".repeat(101))).toBe("display_name_too_long");
  });

  it("returns error key for empty string", () => {
    expect(validateDisplayName("")).toBe("display_name_required");
  });

  it("returns error key for whitespace only", () => {
    expect(validateDisplayName("   ")).toBe("display_name_required");
  });

  it("returns null for name with leading/trailing spaces that trims to valid", () => {
    expect(validateDisplayName("  Alice  ")).toBeNull();
  });
});

describe("validateNewPassword", () => {
  it("returns null for exactly 8 characters", () => {
    expect(validateNewPassword("12345678")).toBeNull();
  });

  it("returns null for more than 8 characters", () => {
    expect(validateNewPassword("averylongpassword")).toBeNull();
  });

  it("returns error key for 7 characters", () => {
    expect(validateNewPassword("1234567")).toBe("password_too_short");
  });

  it("returns error key for empty string", () => {
    expect(validateNewPassword("")).toBe("password_too_short");
  });
});

describe("validateWorkspaceName", () => {
  it("returns null for a regular name", () => {
    expect(validateWorkspaceName("My Workspace")).toBeNull();
  });

  it("returns null for exactly 100 characters", () => {
    expect(validateWorkspaceName("a".repeat(100))).toBeNull();
  });

  it("returns error key for 101 characters", () => {
    expect(validateWorkspaceName("a".repeat(101))).toBe("workspace_name_too_long");
  });

  it("returns error key for empty string", () => {
    expect(validateWorkspaceName("")).toBe("workspace_name_required");
  });

  it("returns error key for whitespace only", () => {
    expect(validateWorkspaceName("   ")).toBe("workspace_name_required");
  });
});

describe("validateSlug", () => {
  it("returns null for a valid slug", () => {
    expect(validateSlug("my-workspace")).toBeNull();
  });

  it("returns null for a single word slug", () => {
    expect(validateSlug("workspace")).toBeNull();
  });

  it("returns null for slug with numbers", () => {
    expect(validateSlug("workspace123")).toBeNull();
  });

  it("returns null for slug starting and ending with letters", () => {
    expect(validateSlug("abc-def-ghi")).toBeNull();
  });

  it("returns error key for invalid chars (uppercase)", () => {
    expect(validateSlug("MyWorkspace")).toBe("slug_invalid");
  });

  it("returns error key for invalid chars (underscore)", () => {
    expect(validateSlug("my_workspace")).toBe("slug_invalid");
  });

  it("returns error key for leading hyphen", () => {
    expect(validateSlug("-workspace")).toBe("slug_invalid");
  });

  it("returns error key for trailing hyphen", () => {
    expect(validateSlug("workspace-")).toBe("slug_invalid");
  });

  it("returns error key for slug longer than 63 chars", () => {
    // The regex requires the middle segment [a-z0-9-]{1,61}, so max total = 2 + 61 = 63
    expect(validateSlug("a" + "-".repeat(61) + "a" + "a")).toBe("slug_invalid"); // 64 chars
  });

  it("returns null for slug exactly at maximum length (63 chars)", () => {
    // 1 start + 61 middle + 1 end = 63
    expect(validateSlug("a" + "b".repeat(61) + "c")).toBeNull();
  });

  it("returns error key for empty string", () => {
    expect(validateSlug("")).toBe("slug_invalid");
  });
});

describe("validateProjectName", () => {
  it("returns null for a regular name", () => {
    expect(validateProjectName("My Project")).toBeNull();
  });

  it("returns null for exactly 200 characters", () => {
    expect(validateProjectName("a".repeat(200))).toBeNull();
  });

  it("returns error key for 201 characters", () => {
    expect(validateProjectName("a".repeat(201))).toBe("project_name_too_long");
  });

  it("returns error key for empty string", () => {
    expect(validateProjectName("")).toBe("project_name_required");
  });

  it("returns error key for whitespace only", () => {
    expect(validateProjectName("   ")).toBe("project_name_required");
  });
});

describe("validateProjectSlugOptional", () => {
  it("returns null for empty string", () => {
    expect(validateProjectSlugOptional("")).toBeNull();
  });

  it("returns null for a valid slug", () => {
    expect(validateProjectSlugOptional("my-project")).toBeNull();
  });

  it("returns slug_invalid for uppercase", () => {
    expect(validateProjectSlugOptional("MyProject")).toBe("slug_invalid");
  });
});

describe("validateRequestingUnit", () => {
  it("returns null for empty (optional)", () => {
    expect(validateRequestingUnit("")).toBeNull();
  });

  it("returns null for a normal value", () => {
    expect(validateRequestingUnit("Marketing")).toBeNull();
  });

  it("returns requesting_unit_too_long for 201 chars", () => {
    expect(validateRequestingUnit("a".repeat(201))).toBe("requesting_unit_too_long");
  });
});

describe("validateDescription", () => {
  it("returns null for empty (optional)", () => {
    expect(validateDescription("")).toBeNull();
  });

  it("returns null for exactly 10000 characters", () => {
    expect(validateDescription("a".repeat(10000))).toBeNull();
  });

  it("returns description_too_long for 10001 chars", () => {
    expect(validateDescription("a".repeat(10001))).toBe("description_too_long");
  });
});

describe("validateProjectDate", () => {
  it("returns null for empty (optional)", () => {
    expect(validateProjectDate("")).toBeNull();
  });

  it("returns null for a valid YYYY-MM-DD", () => {
    expect(validateProjectDate("2026-05-31")).toBeNull();
  });

  it("returns invalid_date for a malformed string", () => {
    expect(validateProjectDate("2026-02-")).toBe("invalid_date");
  });

  it("returns invalid_date for an impossible date (2026-13-40)", () => {
    expect(validateProjectDate("2026-13-40")).toBe("invalid_date");
  });

  it("returns invalid_date for a rollover date (2026-02-30)", () => {
    expect(validateProjectDate("2026-02-30")).toBe("invalid_date");
  });
});

describe("validateDateRange", () => {
  it("returns null when both empty", () => {
    expect(validateDateRange("", "")).toBeNull();
  });

  it("returns null when start <= end", () => {
    expect(validateDateRange("2026-01-01", "2026-12-31")).toBeNull();
  });

  it("returns null when only one is provided", () => {
    expect(validateDateRange("2026-01-01", "")).toBeNull();
    expect(validateDateRange("", "2026-12-31")).toBeNull();
  });

  it("returns end_before_start when end precedes start", () => {
    expect(validateDateRange("2026-12-31", "2026-01-01")).toBe("end_before_start");
  });
});
