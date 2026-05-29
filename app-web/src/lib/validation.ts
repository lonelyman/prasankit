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
