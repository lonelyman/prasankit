export type Lang = "th" | "en";
export const DEFAULT_LANG: Lang = "th";

// ---------------------------------------------------------------------------
// UI strings — labels, buttons, page copy, validation error messages
// ---------------------------------------------------------------------------

const UI_STRINGS: Record<string, Record<Lang, string>> = {
  // --- common ---
  "btn.submit": { th: "ส่ง", en: "Submit" },
  "btn.login": { th: "เข้าสู่ระบบ", en: "Log in" },
  "btn.logout": { th: "ออกจากระบบ", en: "Log out" },
  "btn.signup": { th: "สมัครสมาชิก", en: "Sign up" },
  "btn.resend": { th: "ส่งอีเมลยืนยันอีกครั้ง", en: "Resend verification email" },
  "btn.reset_request": { th: "ส่งลิงก์รีเซ็ตรหัสผ่าน", en: "Send reset link" },
  "btn.reset_confirm": { th: "รีเซ็ตรหัสผ่าน", en: "Reset password" },
  "label.email": { th: "อีเมล", en: "Email" },
  "label.password": { th: "รหัสผ่าน", en: "Password" },
  "label.display_name": { th: "ชื่อที่แสดง", en: "Display name" },
  "label.new_password": { th: "รหัสผ่านใหม่", en: "New password" },
  "label.confirm_password": { th: "ยืนยันรหัสผ่าน", en: "Confirm password" },
  "lang.toggle": { th: "EN", en: "TH" },

  // --- page titles / headings ---
  "page.login.title": { th: "เข้าสู่ระบบ", en: "Log in" },
  "page.signup.title": { th: "สมัครสมาชิก", en: "Sign up" },
  "page.verify_email.title": { th: "ยืนยันอีเมล", en: "Verify email" },
  "page.reset_password.title": { th: "รีเซ็ตรหัสผ่าน", en: "Reset password" },

  // --- navigation links ---
  "link.to_signup": { th: "ยังไม่มีบัญชี? สมัครสมาชิก", en: "No account? Sign up" },
  "link.to_login": { th: "มีบัญชีแล้ว? เข้าสู่ระบบ", en: "Have an account? Log in" },
  "link.forgot_password": { th: "ลืมรหัสผ่าน?", en: "Forgot password?" },

  // --- messages ---
  "msg.signup_success": {
    th: "สมัครสมาชิกสำเร็จ! กรุณาตรวจสอบอีเมลเพื่อยืนยันบัญชีก่อนเข้าสู่ระบบ",
    en: "Signed up successfully! Please check your email to verify your account before logging in.",
  },
  "msg.verify_success": {
    th: "ยืนยันอีเมลสำเร็จ! คุณสามารถเข้าสู่ระบบได้แล้ว",
    en: "Email verified successfully! You can now log in.",
  },
  "msg.verifying": { th: "กำลังยืนยันอีเมล…", en: "Verifying email…" },
  "msg.verify_missing_token": {
    th: "ไม่พบโทเค็นยืนยัน กรุณาคลิกลิงก์จากอีเมลหรือขอลิงก์ใหม่ด้านล่าง",
    en: "No verification token found. Please click the link in your email or request a new one below.",
  },
  "msg.resend_sent": {
    th: "หากบัญชีนี้มีอยู่ในระบบ เราได้ส่งลิงก์ยืนยันใหม่ไปยังอีเมลของคุณแล้ว",
    en: "If an account with that email exists, we've sent a new verification link.",
  },
  "msg.reset_request_sent": {
    th: "หากบัญชีนี้มีอยู่ในระบบ เราได้ส่งลิงก์รีเซ็ตรหัสผ่านไปยังอีเมลของคุณแล้ว",
    en: "If an account with that email exists, we've sent a password reset link.",
  },
  "msg.reset_success": {
    th: "รีเซ็ตรหัสผ่านสำเร็จ! คุณสามารถเข้าสู่ระบบด้วยรหัสผ่านใหม่ได้แล้ว",
    en: "Password reset successfully! You can now log in with your new password.",
  },
  "msg.passwords_no_match": {
    th: "รหัสผ่านไม่ตรงกัน",
    en: "Passwords do not match.",
  },
  "msg.email_not_verified_hint": {
    th: "บัญชีของคุณยังไม่ได้ยืนยันอีเมล กรุณาตรวจสอบกล่องจดหมายหรือขอลิงก์ใหม่",
    en: "Your account email has not been verified. Please check your inbox or request a new link.",
  },
  "msg.loading": { th: "กำลังโหลด…", en: "Loading…" },
  "msg.logged_in_as": { th: "เข้าสู่ระบบในฐานะ", en: "Logged in as" },
  "msg.resend_email_label": {
    th: "ขอลิงก์ยืนยันอีเมลใหม่",
    en: "Request a new verification link",
  },
  "msg.reset_request_label": {
    th: "กรอกอีเมลของคุณเพื่อรับลิงก์รีเซ็ตรหัสผ่าน",
    en: "Enter your email to receive a password reset link.",
  },

  // --- validation error keys (returned from validation.ts) ---
  "invalid_email": { th: "กรุณากรอกอีเมลที่ถูกต้อง", en: "Please enter a valid email address." },
  "password_too_short": { th: "รหัสผ่านต้องมีอย่างน้อย 8 ตัวอักษร", en: "Password must be at least 8 characters." },
  "display_name_required": { th: "กรุณากรอกชื่อที่แสดง", en: "Display name is required." },
  "display_name_too_long": { th: "ชื่อที่แสดงต้องไม่เกิน 100 ตัวอักษร", en: "Display name must be 100 characters or fewer." },
};

export function t(key: string, lang: Lang): string {
  const entry = UI_STRINGS[key];
  if (!entry) return key;
  return entry[lang] ?? key;
}

// ---------------------------------------------------------------------------
// API error code → bilingual message
// ---------------------------------------------------------------------------

const ERROR_MESSAGES: Record<string, Record<Lang, string>> = {
  "auth.email_taken": {
    th: "อีเมลนี้ถูกใช้งานแล้ว",
    en: "This email is already in use.",
  },
  "auth.invalid_credentials": {
    th: "อีเมลหรือรหัสผ่านไม่ถูกต้อง",
    en: "Invalid email or password.",
  },
  "auth.account_locked": {
    th: "บัญชีของคุณถูกล็อค กรุณาติดต่อผู้ดูแลระบบ",
    en: "Your account has been locked. Please contact support.",
  },
  "auth.email_not_verified": {
    th: "บัญชีของคุณยังไม่ได้ยืนยันอีเมล",
    en: "Your email address has not been verified.",
  },
  "auth.token_invalid": {
    th: "ลิงก์ไม่ถูกต้องหรือใช้งานไปแล้ว",
    en: "This link is invalid or has already been used.",
  },
  "auth.token_expired": {
    th: "ลิงก์หมดอายุแล้ว กรุณาขอลิงก์ใหม่",
    en: "This link has expired. Please request a new one.",
  },
  "auth.unauthenticated": {
    th: "กรุณาเข้าสู่ระบบก่อน",
    en: "Please log in to continue.",
  },
  "validation.invalid_input": {
    th: "ข้อมูลที่กรอกไม่ถูกต้อง กรุณาตรวจสอบและลองอีกครั้ง",
    en: "Some fields are invalid. Please check and try again.",
  },
};

const FALLBACK: Record<Lang, string> = {
  th: "เกิดข้อผิดพลาด กรุณาลองอีกครั้ง",
  en: "An error occurred. Please try again.",
};

export function errorMessage(code: string, lang: Lang): string {
  const entry = ERROR_MESSAGES[code];
  if (!entry) return FALLBACK[lang];
  return entry[lang] ?? FALLBACK[lang];
}
