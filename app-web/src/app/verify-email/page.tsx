"use client";

import { Suspense, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import Link from "next/link";
import { useLang } from "@/lib/lang-context";
import { t, errorMessage } from "@/lib/i18n";
import { validateEmail } from "@/lib/validation";
import { apiPost, ApiError } from "@/lib/api";
import { Input, Button, Alert } from "@/components/ui";

// ---------------------------------------------------------------------------
// Inner component that uses useSearchParams() — must be wrapped in <Suspense>
// ---------------------------------------------------------------------------

function VerifyEmailInner() {
  const { lang } = useLang();
  const searchParams = useSearchParams();
  const token = searchParams.get("token");

  type Phase =
    | "verifying"
    | "success"
    | "invalid"
    | "expired"
    | "missing-token";

  const [phase, setPhase] = useState<Phase>(token ? "verifying" : "missing-token");

  // Resend form state
  const [resendEmail, setResendEmail] = useState("");
  const [resendEmailError, setResendEmailError] = useState<string | null>(null);
  const [resendLoading, setResendLoading] = useState(false);
  const [resendDone, setResendDone] = useState(false);

  useEffect(() => {
    if (!token) return;

    async function verify() {
      try {
        await apiPost("/api/v1/auth/verify-email", { token });
        setPhase("success");
      } catch (err) {
        if (err instanceof ApiError) {
          if (err.code === "auth.token_expired") {
            setPhase("expired");
          } else {
            // 404 token_invalid or anything else
            setPhase("invalid");
          }
        } else {
          setPhase("invalid");
        }
      }
    }

    void verify();
  }, [token]);

  async function handleResend(e: React.FormEvent) {
    e.preventDefault();
    setResendEmailError(null);
    const emailErr = validateEmail(resendEmail);
    if (emailErr) {
      setResendEmailError(t(emailErr, lang));
      return;
    }
    setResendLoading(true);
    try {
      await apiPost("/api/v1/auth/verify-email/resend", { email: resendEmail });
    } catch {
      // Anti-enumeration: swallow errors
    } finally {
      setResendLoading(false);
      setResendDone(true);
    }
  }

  const showResendForm = phase === "invalid" || phase === "expired" || phase === "missing-token";

  return (
    <div className="flex-1 flex items-center justify-center p-8">
      <div className="bg-white rounded-2xl shadow-sm border border-zinc-200 w-full max-w-sm p-8 flex flex-col gap-6">
        <h1 className="text-xl font-semibold text-zinc-800">
          {t("page.verify_email.title", lang)}
        </h1>

        {phase === "verifying" && (
          <div className="flex items-center gap-3 text-zinc-500 text-sm">
            <span className="h-4 w-4 rounded-full border-2 border-zinc-400 border-t-transparent animate-spin" />
            {t("msg.verifying", lang)}
          </div>
        )}

        {phase === "success" && (
          <>
            <Alert variant="success">{t("msg.verify_success", lang)}</Alert>
            <Link href="/login" className="text-sm text-zinc-500 hover:text-zinc-800 underline">
              {t("link.to_login", lang)}
            </Link>
          </>
        )}

        {phase === "missing-token" && (
          <Alert variant="info">{t("msg.verify_missing_token", lang)}</Alert>
        )}

        {(phase === "invalid" || phase === "expired") && (
          <Alert variant="error">
            {errorMessage(
              phase === "expired" ? "auth.token_expired" : "auth.token_invalid",
              lang
            )}
          </Alert>
        )}

        {showResendForm && (
          <div className="flex flex-col gap-3">
            <p className="text-sm text-zinc-600">{t("msg.resend_email_label", lang)}</p>
            {resendDone ? (
              <Alert variant="info">{t("msg.resend_sent", lang)}</Alert>
            ) : (
              <form onSubmit={handleResend} className="flex flex-col gap-3" noValidate>
                <Input
                  id="resend-email"
                  type="email"
                  label={t("label.email", lang)}
                  value={resendEmail}
                  onChange={(e) => setResendEmail(e.target.value)}
                  error={resendEmailError ?? undefined}
                  autoComplete="email"
                  disabled={resendLoading}
                />
                <Button type="submit" loading={resendLoading} variant="secondary">
                  {t("btn.resend", lang)}
                </Button>
              </form>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Page export — wraps inner component in Suspense (required for useSearchParams)
// ---------------------------------------------------------------------------

export default function VerifyEmailPage() {
  const { lang } = useLang();
  return (
    <Suspense
      fallback={
        <div className="flex-1 flex items-center justify-center text-zinc-500 text-sm">
          {t("msg.loading", lang)}
        </div>
      }
    >
      <VerifyEmailInner />
    </Suspense>
  );
}
