"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useLang } from "@/lib/lang-context";
import { useWorkspace } from "@/lib/workspace-context";
import { useAuth } from "@/lib/auth-context";
import { t } from "@/lib/i18n";

export function SiteHeader() {
  const { lang, toggleLang } = useLang();
  const { activeSlug, clearActiveSlug } = useWorkspace();
  const { status, logout } = useAuth();
  const router = useRouter();

  async function handleLogout() {
    try {
      await logout();
    } finally {
      clearActiveSlug();
      router.push("/login");
    }
  }

  return (
    <header className="flex items-center justify-between px-6 py-3 border-b border-zinc-200 bg-white">
      <div className="flex items-center gap-3">
        <Link
          href={activeSlug ? "/projects" : "/"}
          className="text-sm font-semibold text-zinc-800 tracking-tight"
        >
          Prasankit
        </Link>
        {activeSlug && (
          <>
            <span className="text-zinc-300">/</span>
            {/* workspace switcher: shows the active workspace; click to switch or create
                (→ /workspaces keeps activeSlug, so no auto-enter bounce; create lives there) */}
            <Link
              href="/workspaces"
              className="text-sm font-medium text-zinc-600 hover:text-zinc-900 transition"
              title={t("btn.switch_workspace", lang)}
            >
              {activeSlug}
            </Link>
          </>
        )}
      </div>
      <div className="flex items-center gap-4">
        {activeSlug && (
          <Link
            href="/projects"
            className="text-sm font-medium text-zinc-600 hover:text-zinc-900 transition"
          >
            {t("nav.projects", lang)}
          </Link>
        )}
        <button
          onClick={toggleLang}
          className="text-xs font-medium text-zinc-500 hover:text-zinc-800 transition px-2 py-1 rounded border border-zinc-200 hover:border-zinc-400"
          aria-label="Toggle language"
        >
          {t("lang.toggle", lang)}
        </button>
        {status === "authenticated" && (
          <button
            onClick={handleLogout}
            className="text-sm font-medium text-zinc-500 hover:text-zinc-800 transition"
          >
            {t("btn.logout", lang)}
          </button>
        )}
      </div>
    </header>
  );
}
