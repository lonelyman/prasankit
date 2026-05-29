import { describe, it, expect } from "vitest";
import { errorMessage, t, DEFAULT_LANG } from "./i18n";

describe("DEFAULT_LANG", () => {
  it("is 'th'", () => {
    expect(DEFAULT_LANG).toBe("th");
  });
});

describe("errorMessage — known codes", () => {
  const knownCodes = [
    "auth.email_taken",
    "auth.invalid_credentials",
    "auth.account_locked",
    "auth.email_not_verified",
    "auth.token_invalid",
    "auth.token_expired",
    "auth.unauthenticated",
    "validation.invalid_input",
  ] as const;

  for (const code of knownCodes) {
    it(`returns non-empty string for '${code}' in 'th'`, () => {
      const msg = errorMessage(code, "th");
      expect(msg).toBeTruthy();
      expect(typeof msg).toBe("string");
    });

    it(`returns non-empty string for '${code}' in 'en'`, () => {
      const msg = errorMessage(code, "en");
      expect(msg).toBeTruthy();
      expect(typeof msg).toBe("string");
    });
  }
});

describe("errorMessage — unknown code", () => {
  it("returns a non-empty fallback string in 'th'", () => {
    const msg = errorMessage("some.unknown.code", "th");
    expect(msg).toBeTruthy();
    expect(typeof msg).toBe("string");
  });

  it("returns a non-empty fallback string in 'en'", () => {
    const msg = errorMessage("some.unknown.code", "en");
    expect(msg).toBeTruthy();
    expect(typeof msg).toBe("string");
  });
});

describe("t — UI strings", () => {
  it("returns the Thai label for a known key", () => {
    const result = t("btn.login", "th");
    expect(result).toBe("เข้าสู่ระบบ");
  });

  it("returns the English label for a known key", () => {
    const result = t("btn.login", "en");
    expect(result).toBe("Log in");
  });

  it("returns the key itself for an unknown key", () => {
    const result = t("nonexistent.key.xyz", "en");
    expect(result).toBe("nonexistent.key.xyz");
  });

  it("resolves validation error keys returned from validation helpers", () => {
    // These keys come from validateEmail / validatePassword etc.
    expect(t("invalid_email", "en")).toBeTruthy();
    expect(t("password_too_short", "en")).toBeTruthy();
    expect(t("display_name_required", "en")).toBeTruthy();
    expect(t("display_name_too_long", "en")).toBeTruthy();
  });
});
