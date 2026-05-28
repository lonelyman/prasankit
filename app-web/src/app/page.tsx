"use client";

import { useEffect, useState } from "react";
import { apiGet, ApiError } from "@/lib/api";

interface HealthData {
  status: string;
  checks: Record<string, string>;
}

type State =
  | { phase: "loading" }
  | { phase: "success"; data: HealthData }
  | { phase: "error"; message: string; code?: string };

export default function Home() {
  const [state, setState] = useState<State>({ phase: "loading" });

  useEffect(() => {
    apiGet<HealthData>("/api/v1/health/ready")
      .then((data) => setState({ phase: "success", data }))
      .catch((err: unknown) => {
        if (err instanceof ApiError) {
          setState({ phase: "error", message: err.message, code: err.code });
        } else if (err instanceof Error) {
          setState({ phase: "error", message: err.message });
        } else {
          setState({ phase: "error", message: "An unexpected error occurred" });
        }
      });
  }, []);

  return (
    <div className="min-h-screen bg-zinc-50 flex items-center justify-center p-8">
      <div className="bg-white rounded-2xl shadow-sm border border-zinc-200 w-full max-w-md p-8">
        <h1 className="text-xl font-semibold text-zinc-800 mb-6">
          Prasankit — Backend Health
        </h1>

        {state.phase === "loading" && (
          <div className="flex items-center gap-3 text-zinc-500">
            <div className="w-4 h-4 border-2 border-zinc-300 border-t-zinc-600 rounded-full animate-spin" />
            <span>Checking backend…</span>
          </div>
        )}

        {state.phase === "success" && (
          <div>
            <div className="flex items-center gap-2 mb-4">
              <span className="text-green-600 font-medium">API:</span>
              <span className="text-green-600">{state.data.status}</span>
              <span className="ml-auto inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                ok
              </span>
            </div>
            <ul className="divide-y divide-zinc-100">
              {Object.entries(state.data.checks).map(([name, status]) => (
                <li
                  key={name}
                  className="flex items-center justify-between py-2"
                >
                  <span className="text-zinc-700 capitalize">{name}</span>
                  <span
                    className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                      status === "ok"
                        ? "bg-green-100 text-green-800"
                        : "bg-red-100 text-red-800"
                    }`}
                  >
                    {status}
                  </span>
                </li>
              ))}
            </ul>
          </div>
        )}

        {state.phase === "error" && (
          <div className="rounded-lg bg-red-50 border border-red-200 p-4">
            <p className="text-red-700 font-medium text-sm">
              Backend unreachable
            </p>
            {state.code && (
              <p className="text-red-500 text-xs mt-1 font-mono">
                {state.code}
              </p>
            )}
            <p className="text-red-600 text-sm mt-1">{state.message}</p>
          </div>
        )}
      </div>
    </div>
  );
}
