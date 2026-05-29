"use client";

// TODO(5b-FE): expand this into the workspace home with workspace listing + create workspace.
// For now this is a protected landing placeholder that confirms auth state.

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";
import { useLang } from "@/lib/lang-context";
import { t } from "@/lib/i18n";
import { Button } from "@/components/ui";

export default function Home() {
  const { status, account, logout } = useAuth();
  const { lang } = useLang();
  const router = useRouter();

  useEffect(() => {
    if (status === "anonymous") {
      router.push("/login");
    }
  }, [status, router]);

  if (status === "loading") {
    return (
      <div className="flex-1 flex items-center justify-center text-zinc-500 text-sm">
        {t("msg.loading", lang)}
      </div>
    );
  }

  if (status === "anonymous") {
    // Redirect is in progress; show nothing to avoid flicker
    return null;
  }

  async function handleLogout() {
    await logout();
    router.push("/login");
  }

  return (
    <div className="flex-1 flex items-center justify-center p-8">
      <div className="bg-white rounded-2xl shadow-sm border border-zinc-200 w-full max-w-sm p-8 flex flex-col gap-4">
        <p className="text-sm text-zinc-500">{t("msg.logged_in_as", lang)}</p>
        <p className="font-semibold text-zinc-800">{account?.display_name}</p>
        <p className="text-sm text-zinc-500">{account?.primary_email}</p>
        <Button variant="secondary" onClick={handleLogout}>
          {t("btn.logout", lang)}
        </Button>
      </div>
    </div>
  );
}
