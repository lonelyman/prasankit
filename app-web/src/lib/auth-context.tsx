"use client";

import React, { createContext, useContext, useEffect, useState } from "react";
import { apiGet, apiPost } from "./api";
import type { Account } from "./types";

interface AuthState {
  status: "loading" | "authenticated" | "anonymous";
  account: Account | null;
}

interface AuthContextValue extends AuthState {
  login(email: string, password: string): Promise<Account>;
  logout(): Promise<void>;
  refresh(): Promise<void>;
}

const AuthContext = createContext<AuthContextValue>({
  status: "loading",
  account: null,
  login: async () => {
    throw new Error("AuthProvider not mounted");
  },
  logout: async () => {},
  refresh: async () => {},
});

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<AuthState>({
    status: "loading",
    account: null,
  });

  // Bootstrap: re-usable outside of the mount effect
  async function fetchMe(signal?: AbortSignal): Promise<void> {
    try {
      const account = await apiGet<Account>("/api/v1/auth/me");
      if (signal?.aborted) return;
      setState({ status: "authenticated", account });
    } catch {
      if (signal?.aborted) return;
      // Treat any failure (401 ApiError or network error) as anonymous
      setState({ status: "anonymous", account: null });
    }
  }

  useEffect(() => {
    const controller = new AbortController();
    // Wrapping in an immediately-invoked async callback inside the effect
    // so the lint rule doesn't flag direct setState calls within the effect body.
    const run = async () => {
      try {
        const account = await apiGet<Account>("/api/v1/auth/me");
        if (!controller.signal.aborted) {
          setState({ status: "authenticated", account });
        }
      } catch {
        if (!controller.signal.aborted) {
          setState({ status: "anonymous", account: null });
        }
      }
    };
    void run();
    return () => controller.abort();
  }, []);

  async function login(email: string, password: string): Promise<Account> {
    // Throws ApiError so the page can handle specific codes
    const account = await apiPost<Account>("/api/v1/auth/login", {
      email,
      password,
    });
    // POST /auth/login always returns the account on success
    setState({ status: "authenticated", account: account! });
    return account!;
  }

  async function logout(): Promise<void> {
    try {
      await apiPost("/api/v1/auth/logout", {});
    } catch {
      // Best-effort — clear local state regardless
    }
    setState({ status: "anonymous", account: null });
  }

  async function refresh(): Promise<void> {
    await fetchMe();
  }

  return (
    <AuthContext.Provider value={{ ...state, login, logout, refresh }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  return useContext(AuthContext);
}
