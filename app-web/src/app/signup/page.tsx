"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useAuth } from "@/lib/auth-context";
import { useLang } from "@/lib/lang-context";
import { t, errorMessage } from "@/lib/i18n";
import { validateEmail, validatePassword, validateConfirmPassword, validateDisplayName } from "@/lib/validation";
import { apiPost, ApiError } from "@/lib/api";
import type { Account } from "@/lib/types";
import { Input, Button, Alert } from "@/components/ui";

export default function SignupPage() {
  const { status } = useAuth();
  const { lang } = useLang();
  const router = useRouter();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [fieldErrors, setFieldErrors] = useState<{
    email?: string;
    password?: string;
    confirm_password?: string;
    display_name?: string;
  }>({});
  const [apiErrorCode, setApiErrorCode] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [done, setDone] = useState(false);

  // Redirect authenticated users away
  useEffect(() => {
    if (status === "authenticated") {
      router.push("/");
    }
  }, [status, router]);

  if (status === "loading" || status === "authenticated") {
    return null;
  }

  if (done) {
    return (
      <div className="flex-1 flex items-center justify-center p-8">
        <div className="bg-white rounded-2xl shadow-sm border border-slate-200 w-full max-w-sm p-8 flex flex-col gap-4">
          <Alert variant="success">{t("msg.signup_success", lang)}</Alert>
          <Link href="/login" className="text-sm text-slate-500 hover:text-slate-800 underline">
            {t("link.to_login", lang)}
          </Link>
        </div>
      </div>
    );
  }

  function validate(): boolean {
    const errors: { email?: string; password?: string; confirm_password?: string; display_name?: string } = {};
    const emailErr = validateEmail(email);
    if (emailErr) errors.email = t(emailErr, lang);
    const pwErr = validatePassword(password);
    if (pwErr) errors.password = t(pwErr, lang);
    const confirmErr = validateConfirmPassword(password, confirmPassword);
    if (confirmErr) errors.confirm_password = t(confirmErr, lang);
    const nameErr = validateDisplayName(displayName);
    if (nameErr) errors.display_name = t(nameErr, lang);
    setFieldErrors(errors);
    return Object.keys(errors).length === 0;
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setApiErrorCode(null);
    if (!validate()) return;

    setLoading(true);
    try {
      await apiPost<Account>("/api/v1/auth/signup", {
        email,
        password,
        display_name: displayName,
      });
      setDone(true);
    } catch (err) {
      if (err instanceof ApiError) {
        setApiErrorCode(err.code);

        // Surface per-field server validation errors
        if (err.code === "validation.invalid_input" && err.details) {
          const serverFieldErrors: Record<string, string> = {};
          for (const detail of err.details) {
            serverFieldErrors[detail.field] = detail.message;
          }
          setFieldErrors((prev) => ({
            ...prev,
            email: serverFieldErrors["email"] ?? prev.email,
            password: serverFieldErrors["password"] ?? prev.password,
            display_name: serverFieldErrors["display_name"] ?? prev.display_name,
          }));
          // Don't show a global error if we already showed field-level ones
          setApiErrorCode(null);
        }
      } else {
        setApiErrorCode("UNKNOWN_ERROR");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex-1 flex items-center justify-center p-8">
      <div className="bg-white rounded-2xl shadow-sm border border-slate-200 w-full max-w-sm p-8 flex flex-col gap-6">
        <h1 className="text-xl font-semibold text-slate-800">
          {t("page.signup.title", lang)}
        </h1>

        <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
          <Input
            id="display_name"
            type="text"
            label={t("label.display_name", lang)}
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            error={fieldErrors.display_name}
            autoComplete="name"
            disabled={loading}
          />
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
          <Input
            id="password"
            type="password"
            label={t("label.password", lang)}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            error={fieldErrors.password}
            autoComplete="new-password"
            disabled={loading}
          />
          <Input
            id="confirm_password"
            type="password"
            label={t("label.confirm_password", lang)}
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            error={fieldErrors.confirm_password}
            autoComplete="new-password"
            disabled={loading}
          />

          {apiErrorCode && (
            <Alert variant="error">{errorMessage(apiErrorCode, lang)}</Alert>
          )}

          <Button type="submit" loading={loading}>
            {t("btn.signup", lang)}
          </Button>
        </form>

        <div className="text-sm text-slate-500">
          <Link href="/login" className="hover:text-slate-800 underline">
            {t("link.to_login", lang)}
          </Link>
        </div>
      </div>
    </div>
  );
}
