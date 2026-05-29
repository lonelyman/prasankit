"use client";

import { useLang } from "@/lib/lang-context";
import { t } from "@/lib/i18n";

export function SiteHeader() {
  const { lang, toggleLang } = useLang();

  return (
    <header className="flex items-center justify-between px-6 py-3 border-b border-zinc-200 bg-white">
      <span className="text-sm font-semibold text-zinc-800 tracking-tight">
        Prasankit
      </span>
      <button
        onClick={toggleLang}
        className="text-xs font-medium text-zinc-500 hover:text-zinc-800 transition px-2 py-1 rounded border border-zinc-200 hover:border-zinc-400"
        aria-label="Toggle language"
      >
        {t("lang.toggle", lang)}
      </button>
    </header>
  );
}
