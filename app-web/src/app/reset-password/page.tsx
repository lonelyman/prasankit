"use client";

import { Suspense, useState } from "react";
import { useSearchParams } from "next/navigation";
import Link from "next/link";
import { useLang } from "@/lib/lang-context";
import { t, errorMessage } from "@/lib/i18n";
import { validateEmail, validateNewPassword } from "@/lib/validation";
import { apiPost, ApiError } from "@/lib/api";
import { Input, Button, Alert } from "@/components/ui";

// ---------------------------------------------------------------------------
// Inner component that uses useSearchParams() — must be wrapped in <Suspense>
// ---------------------------------------------------------------------------

function ResetPasswordInner() {
  const { lang } = useLang();
  const searchParams = useSearchParams();
  const token = searchParams.get("token");

  // --- Request form state (no token) ---
  const [reqEmail, setReqEmail] = useState("");
  const [reqEmailError, setReqEmailError] = useState<string | null>(null);
  const [reqLoading, setReqLoading] = useState(false);
  const [reqDone, setReqDone] = useState(false);

  // --- Confirm form state (token present) ---
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [confirmFieldErrors, setConfirmFieldErrors] = useState<{
    new_password?: string;
    confirm_password?: string;
  }>({});
  const [confirmApiErrorCode, setConfirmApiErrorCode] = useState<string | null>(null);
  const [confirmLoading, setConfirmLoading] = useState(false);
  const [confirmDone, setConfirmDone] = useState(false);

  // ---- Request form ----
  async function handleRequest(e: React.FormEvent) {
    e.preventDefault();
    setReqEmailError(null);
    const emailErr = validateEmail(reqEmail);
    if (emailErr) {
      setReqEmailError(t(emailErr, lang));
      return;
    }
    setReqLoading(true);
    try {
      await apiPost("/api/v1/auth/password-reset/request", { email: reqEmail });
    } catch {
      // Anti-enumeration: always show success
    } finally {
      setReqLoading(false);
      setReqDone(true);
    }
  }

  // ---- Confirm form ----
  function validateConfirm(): boolean {
    const errors: { new_password?: string; confirm_password?: string } = {};
    const pwErr = validateNewPassword(newPassword);
    if (pwErr) errors.new_password = t(pwErr, lang);
    if (confirmPassword !== newPassword) {
      errors.confirm_password = t("msg.passwords_no_match", lang);
    }
    setConfirmFieldErrors(errors);
    return Object.keys(errors).length === 0;
  }

  async function handleConfirm(e: React.FormEvent) {
    e.preventDefault();
    setConfirmApiErrorCode(null);
    if (!validateConfirm()) return;

    setConfirmLoading(true);
    try {
      await apiPost("/api/v1/auth/password-reset/confirm", {
        token,
        new_password: newPassword,
      });
      setConfirmDone(true);
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.code === "validation.invalid_input" && err.details) {
          const serverFieldErrors: Record<string, string> = {};
          for (const detail of err.details) {
            serverFieldErrors[detail.field] = detail.message;
          }
          setConfirmFieldErrors((prev) => ({
            ...prev,
            new_password: serverFieldErrors["new_password"] ?? prev.new_password,
          }));
        } else {
          setConfirmApiErrorCode(err.code);
        }
      } else {
        setConfirmApiErrorCode("UNKNOWN_ERROR");
      }
    } finally {
      setConfirmLoading(false);
    }
  }

  return (
    <div className="flex-1 flex items-center justify-center p-8">
      <div className="bg-white rounded-2xl shadow-sm border border-zinc-200 w-full max-w-sm p-8 flex flex-col gap-6">
        <h1 className="text-xl font-semibold text-zinc-800">
          {t("page.reset_password.title", lang)}
        </h1>

        {/* ---- No token: request form ---- */}
        {!token && (
          <>
            {reqDone ? (
              <>
                <Alert variant="info">{t("msg.reset_request_sent", lang)}</Alert>
                <Link href="/login" className="text-sm text-zinc-500 hover:text-zinc-800 underline">
                  {t("link.to_login", lang)}
                </Link>
              </>
            ) : (
              <form onSubmit={handleRequest} className="flex flex-col gap-4" noValidate>
                <p className="text-sm text-zinc-600">{t("msg.reset_request_label", lang)}</p>
                <Input
                  id="reset-email"
                  type="email"
                  label={t("label.email", lang)}
                  value={reqEmail}
                  onChange={(e) => setReqEmail(e.target.value)}
                  error={reqEmailError ?? undefined}
                  autoComplete="email"
                  disabled={reqLoading}
                />
                <Button type="submit" loading={reqLoading}>
                  {t("btn.reset_request", lang)}
                </Button>
              </form>
            )}
          </>
        )}

        {/* ---- Token present: confirm form ---- */}
        {token && (
          <>
            {confirmDone ? (
              <>
                <Alert variant="success">{t("msg.reset_success", lang)}</Alert>
                <Link href="/login" className="text-sm text-zinc-500 hover:text-zinc-800 underline">
                  {t("link.to_login", lang)}
                </Link>
              </>
            ) : (
              <form onSubmit={handleConfirm} className="flex flex-col gap-4" noValidate>
                <Input
                  id="new-password"
                  type="password"
                  label={t("label.new_password", lang)}
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  error={confirmFieldErrors.new_password}
                  autoComplete="new-password"
                  disabled={confirmLoading}
                />
                <Input
                  id="confirm-password"
                  type="password"
                  label={t("label.confirm_password", lang)}
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  error={confirmFieldErrors.confirm_password}
                  autoComplete="new-password"
                  disabled={confirmLoading}
                />

                {confirmApiErrorCode && (
                  <Alert variant="error">
                    {errorMessage(confirmApiErrorCode, lang)}
                  </Alert>
                )}

                <Button type="submit" loading={confirmLoading}>
                  {t("btn.reset_confirm", lang)}
                </Button>
              </form>
            )}
          </>
        )}

        <Link href="/login" className="text-sm text-zinc-500 hover:text-zinc-800 underline">
          {t("link.to_login", lang)}
        </Link>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Page export — wraps inner component in Suspense (required for useSearchParams)
// ---------------------------------------------------------------------------

export default function ResetPasswordPage() {
  const { lang } = useLang();
  return (
    <Suspense
      fallback={
        <div className="flex-1 flex items-center justify-center text-zinc-500 text-sm">
          {t("msg.loading", lang)}
        </div>
      }
    >
      <ResetPasswordInner />
    </Suspense>
  );
}
