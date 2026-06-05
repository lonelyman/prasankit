import { describe, it, expect } from "vitest";
import { errorMessage, t, DEFAULT_LANG } from "./i18n";
import { PROJECT_STATUS_CODES, PROJECT_TYPE_CODES } from "./project-masters";

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
    // Workspace / tenant / invitation codes (5b-FE)
    "workspace.slug_reserved",
    "workspace.slug_taken",
    "tenant.workspace_required",
    "tenant.workspace_not_found",
    "tenant.forbidden",
    "tenant.permission_denied",
    "invitation.role_not_allowed",
    "invitation.already_member",
    "invitation.already_pending",
    "invitation.invalid",
    "invitation.expired",
    "invitation.email_mismatch",
    // Project codes (M2 FE-A)
    "project.slug_taken",
    "project.invalid_master_code",
    "project.not_found",
    // Positions codes (M2 FE-C)
    "project_position.code_taken",
    "company_position.code_taken",
    "project_position.code_immutable",
    "company_position.code_immutable",
    "project_position.system_immutable",
    "company_position.system_immutable",
    "project_position.invalid_status",
    "company_position.invalid_status",
    "project_position.not_found",
    "company_position.not_found",
    "internal.unexpected",
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

describe("t — project status/type labels", () => {
  // Seed values pinned to migration 000009 (label_th / label_en).
  const STATUS_SEED: Record<string, { th: string; en: string }> = {
    draft: { th: "ฉบับร่าง", en: "Draft" },
    planning: { th: "วางแผน", en: "Planning" },
    proposal: { th: "เสนอราคา", en: "Proposal" },
    active: { th: "กำลังดำเนินการ", en: "Active" },
    closing: { th: "กำลังปิดงาน", en: "Closing" },
    maintenance: { th: "ดูแลรักษา", en: "Maintenance" },
    closed: { th: "ปิดโครงการ", en: "Closed" },
    archived: { th: "เก็บถาวร", en: "Archived" },
  };
  const TYPE_SEED: Record<string, { th: string; en: string }> = {
    internal: { th: "ภายในองค์กร", en: "Internal" },
    client: { th: "งานลูกค้า", en: "Client" },
  };

  it("resolves all 8 status codes in th and en", () => {
    expect(PROJECT_STATUS_CODES.length).toBe(8);
    for (const c of PROJECT_STATUS_CODES) {
      const th = t(`status.${c}`, "th");
      const en = t(`status.${c}`, "en");
      expect(th).not.toBe(`status.${c}`);
      expect(en).not.toBe(`status.${c}`);
      // assert values equal the seed (migration 000009)
      expect(th).toBe(STATUS_SEED[c].th);
      expect(en).toBe(STATUS_SEED[c].en);
    }
  });

  it("resolves both type codes in th and en", () => {
    expect(PROJECT_TYPE_CODES.length).toBe(2);
    for (const c of PROJECT_TYPE_CODES) {
      const th = t(`type.${c}`, "th");
      const en = t(`type.${c}`, "en");
      expect(th).not.toBe(`type.${c}`);
      expect(en).not.toBe(`type.${c}`);
      expect(th).toBe(TYPE_SEED[c].th);
      expect(en).toBe(TYPE_SEED[c].en);
    }
  });

  it("resolves new project validation keys", () => {
    const keys = [
      "project_name_required",
      "project_name_too_long",
      "requesting_unit_too_long",
      "description_too_long",
      "invalid_date",
      "end_before_start",
    ];
    for (const k of keys) {
      for (const lang of ["th", "en"] as const) {
        const v = t(k, lang);
        expect(v).toBeTruthy();
        expect(v).not.toBe(k);
      }
    }
  });
});
