"use client";

import React from "react";

// ---------------------------------------------------------------------------
// Input
// ---------------------------------------------------------------------------

export interface InputProps
  extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
}

export function Input({ label, error, id, className = "", ...props }: InputProps) {
  return (
    <div className="flex flex-col gap-1">
      {label && (
        <label
          htmlFor={id}
          className="text-sm font-medium text-zinc-700"
        >
          {label}
        </label>
      )}
      <input
        id={id}
        className={`rounded-md border px-3 py-2 text-sm outline-none transition focus:ring-2 focus:ring-zinc-400 ${
          error
            ? "border-red-400 bg-red-50 focus:ring-red-300"
            : "border-zinc-300 bg-white"
        } ${className}`}
        {...props}
      />
      {error && <p className="text-xs text-red-600">{error}</p>}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Button
// ---------------------------------------------------------------------------

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  loading?: boolean;
  variant?: "primary" | "secondary" | "ghost";
}

export function Button({
  children,
  loading,
  variant = "primary",
  disabled,
  className = "",
  ...props
}: ButtonProps) {
  const base =
    "inline-flex items-center justify-center rounded-md px-4 py-2 text-sm font-medium transition disabled:opacity-50 disabled:cursor-not-allowed";
  const variants: Record<string, string> = {
    primary: "bg-zinc-900 text-white hover:bg-zinc-700",
    secondary: "bg-zinc-100 text-zinc-900 hover:bg-zinc-200",
    ghost: "text-zinc-600 hover:bg-zinc-100 underline-offset-2 hover:underline",
  };

  return (
    <button
      disabled={disabled || loading}
      className={`${base} ${variants[variant]} ${className}`}
      {...props}
    >
      {loading ? (
        <span className="flex items-center gap-2">
          <span className="h-3.5 w-3.5 rounded-full border-2 border-current border-t-transparent animate-spin" />
          {children}
        </span>
      ) : (
        children
      )}
    </button>
  );
}

// ---------------------------------------------------------------------------
// Alert
// ---------------------------------------------------------------------------

export interface AlertProps {
  variant: "error" | "success" | "info";
  children: React.ReactNode;
  className?: string;
}

const ALERT_STYLES: Record<AlertProps["variant"], string> = {
  error: "bg-red-50 border-red-200 text-red-700",
  success: "bg-green-50 border-green-200 text-green-700",
  info: "bg-blue-50 border-blue-200 text-blue-700",
};

export function Alert({ variant, children, className = "" }: AlertProps) {
  return (
    <div
      role="alert"
      className={`rounded-md border px-4 py-3 text-sm ${ALERT_STYLES[variant]} ${className}`}
    >
      {children}
    </div>
  );
}
