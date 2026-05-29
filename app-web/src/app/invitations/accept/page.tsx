"use client";

import { Suspense, useEffect, useState } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import Link from "next/link";
import { useAuth } from "@/lib/auth-context";
import { useWorkspace } from "@/lib/workspace-context";
import { useLang } from "@/lib/lang-context";
import { t, errorMessage } from "@/lib/i18n";
import { acceptInvitation, listWorkspaces } from "@/lib/workspace-api";
import { ApiError } from "@/lib/api";
import { Alert } from "@/components/ui";

// ---------------------------------------------------------------------------
// Inner component that uses useSearchParams() — must be wrapped in <Suspense>
// ---------------------------------------------------------------------------

type Phase =
  | "accepting"
  | "success"
  | "invalid"
  | "expired"
  | "email-mismatch"
  | "missing-token";

function AcceptInviteInner() {
  const { status } = useAuth();
  const { setActiveSlug } = useWorkspace();
  const { lang } = useLang();
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = searchParams.get("token");

  const [phase, setPhase] = useState<Phase>(token ? "accepting" : "missing-token");
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  // Redirect anonymous users — they must log in first, then re-click the link
  useEffect(() => {
    if (status === "anonymous") {
      router.push("/login");
    }
  }, [status, router]);

  useEffect(() => {
    if (status !== "authenticated" || !token) return;

    async function accept() {
      try {
        const result = await acceptInvitation(token!);
        // Resolve slug from workspace list
        const workspaces = await listWorkspaces();
        const ws = workspaces.find((w) => w.id === result.workspace_id);
        if (ws) {
          setActiveSlug(ws.slug);
        }
        setPhase("success");
        // Brief success state, then redirect
        setTimeout(() => {
          router.push("/");
        }, 1500);
      } catch (err) {
        if (err instanceof ApiError) {
          if (err.code === "invitation.expired") {
            setPhase("expired");
          } else if (err.code === "invitation.email_mismatch") {
            setPhase("email-mismatch");
          } else {
            // invitation.invalid or anything else
            setPhase("invalid");
          }
          setErrorMsg(errorMessage(err.code, lang));
        } else {
          setPhase("invalid");
          setErrorMsg(errorMessage("UNKNOWN_ERROR", lang));
        }
      }
    }

    void accept();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status, token]);

  if (status === "loading") {
    return (
      <div className="flex-1 flex items-center justify-center text-zinc-500 text-sm">
        {t("msg.loading", lang)}
      </div>
    );
  }

  if (status === "anonymous") {
    return null;
  }

  return (
    <div className="flex-1 flex items-center justify-center p-8">
      <div className="bg-white rounded-2xl shadow-sm border border-zinc-200 w-full max-w-sm p-8 flex flex-col gap-6">
        <h1 className="text-xl font-semibold text-zinc-800">
          {t("page.accept_invite.title", lang)}
        </h1>

        {phase === "accepting" && (
          <div className="flex items-center gap-3 text-zinc-500 text-sm">
            <span className="h-4 w-4 rounded-full border-2 border-zinc-400 border-t-transparent animate-spin" />
            {t("msg.accepting_invite", lang)}
          </div>
        )}

        {phase === "success" && (
          <Alert variant="success">{t("msg.accept_invite_success", lang)}</Alert>
        )}

        {phase === "missing-token" && (
          <Alert variant="info">{t("msg.accept_invite_missing_token", lang)}</Alert>
        )}

        {(phase === "invalid" || phase === "expired" || phase === "email-mismatch") && (
          <Alert variant="error">{errorMsg ?? errorMessage("invitation.invalid", lang)}</Alert>
        )}

        {phase !== "accepting" && phase !== "success" && (
          <Link href="/" className="text-sm text-zinc-500 hover:text-zinc-800 underline">
            {t("page.home.title", lang)}
          </Link>
        )}
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Page export — wraps inner component in Suspense (required for useSearchParams)
// ---------------------------------------------------------------------------

export default function AcceptInvitePage() {
  const { lang } = useLang();
  return (
    <Suspense
      fallback={
        <div className="flex-1 flex items-center justify-center text-zinc-500 text-sm">
          {t("msg.loading", lang)}
        </div>
      }
    >
      <AcceptInviteInner />
    </Suspense>
  );
}
