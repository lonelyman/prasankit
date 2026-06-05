"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useAuth } from "@/lib/auth-context";
import { useLang } from "@/lib/lang-context";
import { t, errorMessage } from "@/lib/i18n";
import { validateEmail, validatePassword } from "@/lib/validation";
import { apiPost } from "@/lib/api";
import { ApiError } from "@/lib/api";
import { Input, PasswordInput, Button, Alert } from "@/components/ui";

export default function LoginPage() {
  const { status, login } = useAuth();
  const { lang } = useLang();
  const router = useRouter();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [fieldErrors, setFieldErrors] = useState<{ email?: string; password?: string }>({});
  const [apiErrorCode, setApiErrorCode] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  // Resend verification email state
  const [resendLoading, setResendLoading] = useState(false);
  const [resendDone, setResendDone] = useState(false);

  // Redirect authenticated users away
  useEffect(() => {
    if (status === "authenticated") {
      router.push("/");
    }
  }, [status, router]);

  if (status === "loading" || status === "authenticated") {
    return null;
  }

  function validate(): boolean {
    const errors: { email?: string; password?: string } = {};
    const emailErr = validateEmail(email);
    if (emailErr) errors.email = t(emailErr, lang);
    const pwErr = validatePassword(password);
    if (pwErr) errors.password = t(pwErr, lang);
    setFieldErrors(errors);
    return Object.keys(errors).length === 0;
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setApiErrorCode(null);
    setResendDone(false);
    if (!validate()) return;

    setLoading(true);
    try {
      await login(email, password);
      router.push("/");
    } catch (err) {
      if (err instanceof ApiError) {
        setApiErrorCode(err.code);
      } else {
        setApiErrorCode("UNKNOWN_ERROR");
      }
    } finally {
      setLoading(false);
    }
  }

  async function handleResend() {
    setResendLoading(true);
    try {
      await apiPost("/api/v1/auth/verify-email/resend", { email });
    } catch {
      // Anti-enumeration: never reveal errors
    } finally {
      setResendLoading(false);
      setResendDone(true);
    }
  }

  const showEmailNotVerified = apiErrorCode === "auth.email_not_verified";

  return (
    <div className="flex-1 flex items-center justify-center p-8">
      <div className="bg-white rounded-2xl shadow-sm border border-slate-200 w-full max-w-sm p-8 flex flex-col gap-6">
        <h1 className="text-xl font-semibold text-slate-800">
          {t("page.login.title", lang)}
        </h1>

        <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
          <Input
            id="email"
            type="email"
            label={t("label.email", lang)}
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            error={fieldErrors.email}
            autoComplete="email"
            disabled={loading}
          />
          <PasswordInput
            id="password"
            label={t("label.password", lang)}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            error={fieldErrors.password}
            autoComplete="current-password"
            disabled={loading}
            showPasswordLabel={t("a11y.show_password", lang)}
            hidePasswordLabel={t("a11y.hide_password", lang)}
          />

          {apiErrorCode && !showEmailNotVerified && (
            <Alert variant="error">{errorMessage(apiErrorCode, lang)}</Alert>
          )}

          {showEmailNotVerified && (
            <Alert variant="info">
              <p className="mb-2">{t("msg.email_not_verified_hint", lang)}</p>
              {resendDone ? (
                <p className="text-sm">{t("msg.resend_sent", lang)}</p>
              ) : (
                <Button
                  type="button"
                  variant="secondary"
                  loading={resendLoading}
                  onClick={handleResend}
                  className="mt-1"
                >
                  {t("btn.resend", lang)}
                </Button>
              )}
            </Alert>
          )}

          <Button type="submit" loading={loading}>
            {t("btn.login", lang)}
          </Button>
        </form>

        <div className="flex flex-col gap-2 text-sm text-slate-500">
          <Link href="/signup" className="hover:text-slate-800 underline">
            {t("link.to_signup", lang)}
          </Link>
          <Link href="/reset-password" className="hover:text-slate-800 underline">
            {t("link.forgot_password", lang)}
          </Link>
        </div>
      </div>
    </div>
  );
}
