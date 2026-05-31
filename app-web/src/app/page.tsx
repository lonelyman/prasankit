"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";
import { useWorkspace } from "@/lib/workspace-context";
import { useLang } from "@/lib/lang-context";
import { t } from "@/lib/i18n";

// Home is a pure router: it sends an authenticated user straight to their work area
// (/projects) so login → work is one click. No landing-page hop.
//   - anonymous            → /login
//   - authed, active ws    → /projects   (work area; switch/logout live in the header)
//   - authed, no active ws → /workspaces (pick or create a workspace)
// (Member invitation moved out of the home dashboard — it belongs with team management, FE-B.)
export default function Home() {
  const { status } = useAuth();
  const { activeSlug } = useWorkspace();
  const { lang } = useLang();
  const router = useRouter();

  useEffect(() => {
    if (status === "loading") return;
    if (status === "anonymous") {
      router.replace("/login");
      return;
    }
    // authenticated
    if (activeSlug) {
      router.replace("/projects");
    } else {
      router.replace("/workspaces");
    }
  }, [status, activeSlug, router]);

  return (
    <div className="flex-1 flex items-center justify-center p-8">
      <p className="text-sm text-zinc-500">{t("msg.loading", lang)}</p>
    </div>
  );
}
