"use client";

import React, { createContext, useContext, useState, useEffect } from "react";

const STORAGE_KEY = "prasankit_active_workspace";

interface WorkspaceContextValue {
  activeSlug: string | null;
  setActiveSlug(slug: string): void;
  clearActiveSlug(): void;
}

const WorkspaceContext = createContext<WorkspaceContextValue>({
  activeSlug: null,
  setActiveSlug: () => {},
  clearActiveSlug: () => {},
});

export function WorkspaceProvider({ children }: { children: React.ReactNode }) {
  // SSR-safe lazy initialization: start with null on the server,
  // hydrate from localStorage on the client after mount.
  const [activeSlug, setActiveSlugState] = useState<string | null>(null);

  useEffect(() => {
    const init = () => {
      if (typeof window !== "undefined") {
        const stored = window.localStorage.getItem(STORAGE_KEY);
        if (stored) {
          setActiveSlugState(stored);
        }
      }
    };
    init();
  }, []);

  function setActiveSlug(slug: string): void {
    if (typeof window !== "undefined") {
      window.localStorage.setItem(STORAGE_KEY, slug);
    }
    setActiveSlugState(slug);
  }

  function clearActiveSlug(): void {
    if (typeof window !== "undefined") {
      window.localStorage.removeItem(STORAGE_KEY);
    }
    setActiveSlugState(null);
  }

  return (
    <WorkspaceContext.Provider value={{ activeSlug, setActiveSlug, clearActiveSlug }}>
      {children}
    </WorkspaceContext.Provider>
  );
}

export function useWorkspace(): WorkspaceContextValue {
  return useContext(WorkspaceContext);
}
