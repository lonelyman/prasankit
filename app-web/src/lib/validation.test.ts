import { describe, it, expect } from "vitest";
import {
  validateEmail,
  validatePassword,
  validateDisplayName,
  validateNewPassword,
  validateWorkspaceName,
  validateSlug,
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
