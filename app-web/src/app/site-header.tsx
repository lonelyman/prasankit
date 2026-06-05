"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useLang } from "@/lib/lang-context";
import { useWorkspace } from "@/lib/workspace-context";
import { useAuth } from "@/lib/auth-context";
import { t } from "@/lib/i18n";
import { Dropdown } from "@/components/ui";

export function SiteHeader() {
  const { lang, setLang } = useLang();
  const { activeSlug, activeWorkspace, workspaces, setActiveSlug, clearActiveSlug } =
    useWorkspace();
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

  // Display the workspace NAME; fall back to the raw slug until the list loads
  // (resilient — the switcher still works if listWorkspaces() failed).
  const activeLabel = activeWorkspace?.workspace_name ?? activeSlug;

  function switchTo(slug: string) {
    if (slug !== activeSlug) {
      setActiveSlug(slug);
      router.push("/projects");
    }
  }

  return (
    <header className="flex items-center justify-between px-6 py-3 border-b border-slate-200 bg-white">
      <div className="flex items-center gap-3">
        <Link
          href={activeSlug ? "/projects" : "/"}
          className="text-sm font-semibold text-brand-700 tracking-tight"
        >
          Prasankit
        </Link>
        {activeSlug && (
          <>
            <span className="text-slate-300">/</span>
            {/* Workspace switcher: a real dropdown listing all workspaces.
                Click a row = switch in one click; footer = create a new one. */}
            <Dropdown
              align="left"
              trigger={({ toggle, open }) => (
                <button
                  type="button"
                  onClick={toggle}
                  aria-haspopup="menu"
                  aria-expanded={open}
                  aria-label={t("nav.switch_workspace", lang)}
                  title={t("nav.switch_workspace", lang)}
                  className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-sm font-medium text-slate-600 hover:bg-slate-100 hover:text-slate-900 transition"
                >
                  <span className="max-w-[12rem] truncate">{activeLabel}</span>
                  <span className="text-xs text-slate-400" aria-hidden="true">
                    ▾
                  </span>
                </button>
              )}
            >
              {({ close }) => (
                <>
                  {workspaces.map((ws) => {
                    const isActive = ws.slug === activeSlug;
                    return (
                      <button
                        key={ws.id}
                        type="button"
                        role="menuitem"
                        onClick={() => {
                          close();
                          switchTo(ws.slug);
                        }}
                        className="flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 transition"
                      >
                        <span className="truncate">{ws.workspace_name}</span>
                        {isActive && (
                          <span className="text-slate-500" aria-hidden="true">
                            ✓
                          </span>
                        )}
                      </button>
                    );
                  })}
                  <div className="my-1 border-t border-slate-100" />
                  <button
                    type="button"
                    role="menuitem"
                    onClick={() => {
                      close();
                      router.push("/workspaces");
                    }}
                    className="flex w-full items-center px-3 py-2 text-left text-sm font-medium text-slate-600 hover:bg-slate-100 transition"
                  >
                    {t("btn.new_workspace", lang)}
                  </button>
                </>
              )}
            </Dropdown>
          </>
        )}
      </div>
      <div className="flex items-center gap-4">
        {activeSlug && (
          <Link
            href="/projects"
            className="text-sm font-medium text-slate-600 hover:text-slate-900 transition"
          >
            {t("nav.projects", lang)}
          </Link>
        )}
        {activeSlug && (
          <Link
            href="/settings/positions"
            className="text-sm font-medium text-slate-600 hover:text-slate-900 transition"
          >
            {t("nav.settings", lang)}
          </Link>
        )}
        {/* Segmented language toggle: both options visible, the active one
            highlighted — so the current language is unambiguous (vs a single
            action-label that showed the target language). */}
        <div
          role="group"
          aria-label="Language"
          className="inline-flex items-center rounded-md border border-slate-200 p-0.5 text-xs font-medium"
        >
          {(["th", "en"] as const).map((code) => {
            const active = lang === code;
            return (
              <button
                key={code}
                onClick={() => setLang(code)}
                aria-pressed={active}
                className={`px-2 py-0.5 rounded transition ${
                  active
                    ? "bg-slate-900 text-white"
                    : "text-slate-500 hover:text-slate-800"
                }`}
              >
                {code === "th" ? "ไทย" : "EN"}
              </button>
            );
          })}
        </div>
        {status === "authenticated" && (
          <button
            onClick={handleLogout}
            className="text-sm font-medium text-slate-500 hover:text-slate-800 transition"
          >
            {t("btn.logout", lang)}
          </button>
        )}
      </div>
    </header>
  );
}
