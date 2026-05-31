"use client";

import React from "react";
import Link from "next/link";

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
// Modal — overlay shell factored from ConfirmDialog (same fixed/overlay/Escape
// technique). ConfirmDialog is now a thin wrapper around this; both coexist.
// ---------------------------------------------------------------------------

export interface ModalProps {
  open: boolean;
  title: string;
  onClose(): void;
  children: React.ReactNode;
  className?: string; // override the card width/shape if needed
}

export function Modal({ open, title, onClose, children, className = "w-full max-w-sm" }: ModalProps) {
  React.useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      onClick={onClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        className={`rounded-2xl bg-white p-6 shadow-lg flex flex-col gap-4 ${className}`}
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="text-lg font-semibold text-zinc-800">{title}</h2>
        {children}
      </div>
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
  return (
    <Modal open={open} title={title} onClose={onCancel}>
      {body && <div className="text-sm text-zinc-600">{body}</div>}
      <div className="flex justify-end gap-2">
        <Button variant="ghost" onClick={onCancel} disabled={loading}>
          {cancelLabel}
        </Button>
        <Button variant={confirmVariant} onClick={onConfirm} loading={loading}>
          {confirmLabel}
        </Button>
      </div>
    </Modal>
  );
}

// ---------------------------------------------------------------------------
// Badge — a colored rounded pill (status/type chips). Color is a Tailwind
// class string supplied by the caller (e.g. a FE status→color map).
// ---------------------------------------------------------------------------

export interface BadgeProps {
  children: React.ReactNode;
  /** Tailwind bg+text classes, e.g. "bg-green-100 text-green-700". */
  color?: string;
  className?: string;
}

export function Badge({ children, color = "bg-zinc-100 text-zinc-700", className = "" }: BadgeProps) {
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${color} ${className}`}
    >
      {children}
    </span>
  );
}

// ---------------------------------------------------------------------------
// Breadcrumb — "a / b / c" orientation trail; links where href is present.
// ---------------------------------------------------------------------------

export interface BreadcrumbItem {
  label: string;
  href?: string;
}

export interface BreadcrumbProps {
  items: BreadcrumbItem[];
  className?: string;
}

export function Breadcrumb({ items, className = "" }: BreadcrumbProps) {
  return (
    <nav aria-label="Breadcrumb" className={`flex items-center gap-1.5 text-sm ${className}`}>
      {items.map((item, i) => (
        <React.Fragment key={`${item.label}-${i}`}>
          {i > 0 && <span className="text-zinc-300">/</span>}
          {item.href ? (
            <Link
              href={item.href}
              className="text-zinc-500 hover:text-zinc-900 transition"
            >
              {item.label}
            </Link>
          ) : (
            <span className="text-zinc-800 font-medium" aria-current="page">
              {item.label}
            </span>
          )}
        </React.Fragment>
      ))}
    </nav>
  );
}

// ---------------------------------------------------------------------------
// Dropdown — a lightweight headless popover: a trigger + an absolutely-
// positioned panel below it; closes on outside-click + Escape. Reuses the
// ConfirmDialog keydown technique but is NOT a full-screen modal.
// ---------------------------------------------------------------------------

export interface DropdownProps {
  /** Render the trigger; receives the current open state + a toggle. */
  trigger: (args: { open: boolean; toggle(): void }) => React.ReactNode;
  /** Panel content; `close` lets a row dismiss the menu after acting. */
  children: (args: { close(): void }) => React.ReactNode;
  /** Panel alignment relative to the trigger. */
  align?: "left" | "right";
  className?: string;
}

export function Dropdown({ trigger, children, align = "left", className = "" }: DropdownProps) {
  const [open, setOpen] = React.useState(false);
  const rootRef = React.useRef<HTMLDivElement>(null);

  const close = React.useCallback(() => setOpen(false), []);
  const toggle = React.useCallback(() => setOpen((o) => !o), []);

  React.useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") close();
    };
    const onClick = (e: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) {
        close();
      }
    };
    window.addEventListener("keydown", onKey);
    window.addEventListener("mousedown", onClick);
    return () => {
      window.removeEventListener("keydown", onKey);
      window.removeEventListener("mousedown", onClick);
    };
  }, [open, close]);

  return (
    <div ref={rootRef} className={`relative inline-block ${className}`}>
      {trigger({ open, toggle })}
      {open && (
        <div
          role="menu"
          className={`absolute z-40 mt-1 min-w-[12rem] rounded-md border border-zinc-200 bg-white py-1 shadow-lg ${
            align === "right" ? "right-0" : "left-0"
          }`}
        >
          {children({ close })}
        </div>
      )}
    </div>
  );
}
