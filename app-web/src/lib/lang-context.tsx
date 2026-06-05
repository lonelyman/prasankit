"use client";

import React, { createContext, useContext, useState, useEffect } from "react";
import { type Lang, DEFAULT_LANG } from "./i18n";

const STORAGE_KEY = "prasankit_lang";

interface LangContextValue {
  lang: Lang;
  setLang: (lang: Lang) => void;
  toggleLang: () => void;
}

const LangContext = createContext<LangContextValue>({
  lang: DEFAULT_LANG,
  setLang: () => {},
  toggleLang: () => {},
});

export function LangProvider({ children }: { children: React.ReactNode }) {
  // Start from DEFAULT_LANG so the server render and the first client render agree
  // (no hydration mismatch); hydrate the persisted choice from localStorage after
  // mount. Mirrors the activeSlug pattern in workspace-context.
  const [lang, setLangState] = useState<Lang>(DEFAULT_LANG);

  useEffect(() => {
    const init = () => {
      if (typeof window === "undefined") return;
      const stored = window.localStorage.getItem(STORAGE_KEY);
      if (stored === "th" || stored === "en") {
        setLangState(stored);
      }
    };
    init();
  }, []);

  function setLang(next: Lang) {
    setLangState(next);
    if (typeof window !== "undefined") {
      window.localStorage.setItem(STORAGE_KEY, next);
    }
  }

  function toggleLang() {
    setLang(lang === "th" ? "en" : "th");
  }

  return (
    <LangContext.Provider value={{ lang, setLang, toggleLang }}>
      {children}
    </LangContext.Provider>
  );
}

export function useLang(): LangContextValue {
  return useContext(LangContext);
}
