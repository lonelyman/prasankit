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
  variant?: "primary" | "secondary" | "ghost" | "danger";
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
    danger: "bg-red-600 text-white hover:bg-red-500",
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

// ---------------------------------------------------------------------------
// Select
// ---------------------------------------------------------------------------

export interface SelectOption {
  value: string;
  label: string;
}

export interface SelectProps
  extends React.SelectHTMLAttributes<HTMLSelectElement> {
  label?: string;
  error?: string;
  options: SelectOption[];
}

export function Select({ label, error, id, options, className = "", ...props }: SelectProps) {
  return (
    <div className="flex flex-col gap-1">
      {label && (
        <label htmlFor={id} className="text-sm font-medium text-zinc-700">
          {label}
        </label>
      )}
      <select
        id={id}
        className={`rounded-md border px-3 py-2 text-sm outline-none transition focus:ring-2 focus:ring-zinc-400 bg-white ${
          error
            ? "border-red-400 bg-red-50 focus:ring-red-300"
            : "border-zinc-300"
        } ${className}`}
        {...props}
      >
        {options.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
      {error && <p className="text-xs text-red-600">{error}</p>}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Pagination
// ---------------------------------------------------------------------------

export interface PaginationProps {
  page: number;
  totalPages: number;
  hasMore: boolean;
  onPrev(): void;
  onNext(): void;
  labelPrev: string;
  labelNext: string;
  labelStatus: string; // ALREADY interpolated by the caller — Pagination does no templating
  disabled?: boolean;
}

export function Pagination({
  page,
  totalPages,
  hasMore,
  onPrev,
  onNext,
  labelPrev,
  labelNext,
  labelStatus,
  disabled,
}: PaginationProps) {
  return (
    <div className="flex items-center justify-between">
      <Button
        variant="secondary"
        onClick={onPrev}
        disabled={disabled || page <= 1}
      >
        {labelPrev}
      </Button>
      <span className="text-sm text-zinc-500">{labelStatus}</span>
      <Button
        variant="secondary"
        onClick={onNext}
        disabled={disabled || !hasMore || page >= totalPages}
      >
        {labelNext}
      </Button>
    </div>
  );
}

// ---------------------------------------------------------------------------
// ConfirmDialog
// ---------------------------------------------------------------------------

export interface ConfirmDialogProps {
  open: boolean;
  title: string;
  body?: React.ReactNode;
  confirmLabel: string;
  cancelLabel: string;
  confirmVariant?: "primary" | "danger";
  loading?: boolean;
  onConfirm(): void;
  onCancel(): void;
}

export function ConfirmDialog({
  open,
  title,
  body,
  confirmLabel,
  cancelLabel,
  confirmVariant = "primary",
  loading,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  React.useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onCancel();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onCancel]);

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      onClick={onCancel}
    >
      <div
        role="dialog"
        aria-modal="true"
        className="w-full max-w-sm rounded-2xl bg-white p-6 shadow-lg flex flex-col gap-4"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="text-lg font-semibold text-zinc-800">{title}</h2>
        {body && <div className="text-sm text-zinc-600">{body}</div>}
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={onCancel} disabled={loading}>
            {cancelLabel}
          </Button>
          <Button variant={confirmVariant} onClick={onConfirm} loading={loading}>
            {confirmLabel}
          </Button>
        </div>
      </div>
    </div>
  );
}
