"use client";

import React, { createContext, useContext, useState, useEffect, useCallback } from "react";
import { useAuth } from "./auth-context";
import { listWorkspaces } from "./workspace-api";
import type { WorkspaceWithRole } from "./types";

const STORAGE_KEY = "prasankit_active_workspace";

interface WorkspaceContextValue {
  activeSlug: string | null;
  setActiveSlug(slug: string): void;
  clearActiveSlug(): void;
  // List cache for the header switcher (resilient: empty when the fetch fails).
  workspaces: WorkspaceWithRole[];
  // The cached entry whose slug === activeSlug, or null (use slug as fallback).
  activeWorkspace: WorkspaceWithRole | null;
  // Re-fetch the list (e.g. after creating a workspace).
  refreshWorkspaces(): Promise<void>;
}

const WorkspaceContext = createContext<WorkspaceContextValue>({
  activeSlug: null,
  setActiveSlug: () => {},
  clearActiveSlug: () => {},
  workspaces: [],
  activeWorkspace: null,
  refreshWorkspaces: async () => {},
});

export function WorkspaceProvider({ children }: { children: React.ReactNode }) {
  const { status } = useAuth();

  // SSR-safe lazy initialization: start with null on the server,
  // hydrate from localStorage on the client after mount.
  const [activeSlug, setActiveSlugState] = useState<string | null>(null);
  const [workspaces, setWorkspaces] = useState<WorkspaceWithRole[]>([]);

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

  // Cache the workspace list once authenticated so the header switcher can
  // show display names + the full list without each consumer re-fetching.
  // Resilient: on failure leave the cache empty (the switcher falls back to
  // the raw slug).
  const refreshWorkspaces = useCallback(async () => {
    try {
      const items = await listWorkspaces();
      setWorkspaces(items);
    } catch {
      // swallow — header degrades gracefully to the slug
    }
  }, []);

  useEffect(() => {
    const sync = async () => {
      if (status === "authenticated") {
        await refreshWorkspaces();
      } else if (status === "anonymous") {
        setWorkspaces([]);
      }
    };
    void sync();
  }, [status, refreshWorkspaces]);

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

  const activeWorkspace =
    (activeSlug && workspaces.find((ws) => ws.slug === activeSlug)) || null;

  return (
    <WorkspaceContext.Provider
      value={{
        activeSlug,
        setActiveSlug,
        clearActiveSlug,
        workspaces,
        activeWorkspace,
        refreshWorkspaces,
      }}
    >
      {children}
    </WorkspaceContext.Provider>
  );
}

export function useWorkspace(): WorkspaceContextValue {
  return useContext(WorkspaceContext);
}
