/**
 * Client-side validation helpers mirroring BE validation rules.
 * Returns an i18n error-key string on failure, or null when valid.
 * BE is still authoritative — always surface server errors too.
 */

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function validateEmail(v: string): string | null {
  if (!v.trim() || !EMAIL_RE.test(v.trim())) return "invalid_email";
  return null;
}

export function validatePassword(v: string): string | null {
  if (v.length < 8) return "password_too_short";
  return null;
}

// Confirm-password: must match the password exactly (catches signup typos; FE-only — only `password` is sent to the BE).
export function validateConfirmPassword(password: string, confirm: string): string | null {
  if (confirm !== password) return "password_mismatch";
  return null;
}

export function validateDisplayName(v: string): string | null {
  const trimmed = v.trim();
  if (!trimmed) return "display_name_required";
  if (trimmed.length > 100) return "display_name_too_long";
  return null;
}

export function validateNewPassword(v: string): string | null {
  if (v.length < 8) return "password_too_short";
  return null;
}

const SLUG_RE = /^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$/;

export function validateWorkspaceName(v: string): string | null {
  const trimmed = v.trim();
  if (!trimmed) return "workspace_name_required";
  if (trimmed.length > 100) return "workspace_name_too_long";
  return null;
}

export function validateSlug(v: string): string | null {
  if (!SLUG_RE.test(v)) return "slug_invalid";
  return null;
}

// ---------------------------------------------------------------------------
// Project validators (M2 FE-A)
// BE is authoritative — always surface server errors too. Note Go len() counts
// bytes, JS .length counts UTF-16 units, so Thai text hits the BE byte limit
// sooner; FE check is a soft guard only.
// ---------------------------------------------------------------------------

export function validateProjectName(v: string): string | null {
  const trimmed = v.trim();
  if (!trimmed) return "project_name_required";
  if (trimmed.length > 200) return "project_name_too_long";
  return null;
}

// Slug is OPTIONAL (BE validates the pattern only when non-empty and stores
// NULL when blank). Blank-after-trim → valid (null); otherwise must pass SLUG_RE.
export function validateProjectSlugOptional(v: string): string | null {
  const trimmed = v.trim();
  if (!trimmed) return null;
  if (!SLUG_RE.test(trimmed)) return "slug_invalid";
  return null;
}

export function validateRequestingUnit(v: string): string | null {
  const trimmed = v.trim();
  if (!trimmed) return null; // optional; BE min-1 handled by trim-and-omit at submit
  if (trimmed.length > 200) return "requesting_unit_too_long";
  return null;
}

export function validateDescription(v: string): string | null {
  const trimmed = v.trim();
  if (!trimmed) return null; // optional
  if (trimmed.length > 10000) return "description_too_long";
  return null;
}

const DATE_RE = /^\d{4}-\d{2}-\d{2}$/;

// Optional; if present must be YYYY-MM-DD AND a real calendar date. The UTC
// round-trip rejects rollover dates like 2026-02-30 that `new Date()` silently
// shifts (BE time.Parse("2006-01-02") rejects them).
export function validateProjectDate(v: string): string | null {
  const trimmed = v.trim();
  if (!trimmed) return null;
  if (!DATE_RE.test(trimmed)) return "invalid_date";
  const [y, m, d] = trimmed.split("-").map(Number);
  const dt = new Date(Date.UTC(y, m - 1, d));
  if (
    dt.getUTCFullYear() !== y ||
    dt.getUTCMonth() !== m - 1 ||
    dt.getUTCDate() !== d
  ) {
    return "invalid_date";
  }
  return null;
}

// If both present, start <= end. String compare is valid ONLY for well-formed
// ISO dates — the caller MUST run validateProjectDate on both first and only
// call this when both are structurally valid.
export function validateDateRange(start: string, end: string): string | null {
  const s = start.trim();
  const e = end.trim();
  if (!s || !e) return null;
  if (s > e) return "end_before_start";
  return null;
}
