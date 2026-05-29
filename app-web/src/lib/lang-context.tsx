"use client";

import React, { createContext, useContext, useState } from "react";
import { type Lang, DEFAULT_LANG } from "./i18n";

interface LangContextValue {
  lang: Lang;
  toggleLang: () => void;
}

const LangContext = createContext<LangContextValue>({
  lang: DEFAULT_LANG,
  toggleLang: () => {},
});

export function LangProvider({ children }: { children: React.ReactNode }) {
  const [lang, setLang] = useState<Lang>(DEFAULT_LANG);

  function toggleLang() {
    setLang((prev) => (prev === "th" ? "en" : "th"));
  }

  return (
    <LangContext.Provider value={{ lang, toggleLang }}>
      {children}
    </LangContext.Provider>
  );
}

export function useLang(): LangContextValue {
  return useContext(LangContext);
}
