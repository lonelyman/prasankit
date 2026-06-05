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
          className="text-sm font-medium text-slate-700"
        >
          {label}
        </label>
      )}
      <input
        id={id}
        className={`rounded-md border px-3 py-2 text-sm outline-none transition focus:ring-2 focus:ring-brand-500/40 ${
          error
            ? "border-red-400 bg-red-50 focus:ring-red-300"
            : "border-slate-300 bg-white focus:border-brand-500"
        } ${className}`}
        {...props}
      />
      {error && <p className="text-xs text-red-600">{error}</p>}
    </div>
  );
}

// ---------------------------------------------------------------------------
// PasswordInput — Input with a show/hide eye toggle. aria labels are passed in
// (the kit is i18n-agnostic) and default to English.
// ---------------------------------------------------------------------------

export interface PasswordInputProps extends InputProps {
  showPasswordLabel?: string;
  hidePasswordLabel?: string;
}

export function PasswordInput({
  label,
  error,
  id,
  className = "",
  showPasswordLabel = "Show password",
  hidePasswordLabel = "Hide password",
  ...props
}: PasswordInputProps) {
  const [visible, setVisible] = React.useState(false);
  return (
    <div className="flex flex-col gap-1">
      {label && (
        <label htmlFor={id} className="text-sm font-medium text-slate-700">
          {label}
        </label>
      )}
      <div className="relative">
        <input
          id={id}
          className={`w-full rounded-md border px-3 py-2 pr-10 text-sm outline-none transition focus:ring-2 focus:ring-brand-500/40 ${
            error
              ? "border-red-400 bg-red-50 focus:ring-red-300"
              : "border-slate-300 bg-white focus:border-brand-500"
          } ${className}`}
          {...props}
          type={visible ? "text" : "password"}
        />
        <button
          type="button"
          onClick={() => setVisible((v) => !v)}
          aria-label={visible ? hidePasswordLabel : showPasswordLabel}
          aria-pressed={visible}
          title={visible ? hidePasswordLabel : showPasswordLabel}
          className="absolute inset-y-0 right-0 flex items-center px-3 text-slate-400 hover:text-slate-600 transition-colors cursor-pointer"
        >
          {visible ? <EyeOffIcon /> : <EyeIcon />}
        </button>
      </div>
      {error && <p className="text-xs text-red-600">{error}</p>}
    </div>
  );
}

function EyeIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} className="h-5 w-5" aria-hidden="true">
      <path strokeLinecap="round" strokeLinejoin="round" d="M2.036 12.322a1.012 1.012 0 0 1 0-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178Z" />
      <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z" />
    </svg>
  );
}

function EyeOffIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} className="h-5 w-5" aria-hidden="true">
      <path strokeLinecap="round" strokeLinejoin="round" d="M3.98 8.223A10.477 10.477 0 0 0 1.934 12C3.226 16.338 7.244 19.5 12 19.5c.993 0 1.953-.138 2.863-.395M6.228 6.228A10.45 10.45 0 0 1 12 4.5c4.756 0 8.774 3.162 10.066 7.498a10.523 10.523 0 0 1-4.293 5.774M6.228 6.228 3 3m3.228 3.228 3.65 3.65m7.894 7.894L21 21m-3.228-3.228-3.65-3.65m0 0a3 3 0 1 0-4.243-4.243m4.243 4.243L9.88 9.88" />
    </svg>
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
    "inline-flex items-center justify-center rounded-md px-4 py-2 text-sm font-medium transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500/50 focus-visible:ring-offset-1 disabled:opacity-50 disabled:cursor-not-allowed";
  const variants: Record<string, string> = {
    primary: "bg-brand-600 text-white hover:bg-brand-700",
    secondary: "bg-slate-100 text-slate-900 hover:bg-slate-200",
    ghost: "text-slate-600 hover:bg-slate-100 underline-offset-2 hover:underline",
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
        <label htmlFor={id} className="text-sm font-medium text-slate-700">
          {label}
        </label>
      )}
      <select
        id={id}
        className={`rounded-md border px-3 py-2 text-sm outline-none transition focus:ring-2 focus:ring-brand-500/40 bg-white ${
          error
            ? "border-red-400 bg-red-50 focus:ring-red-300"
            : "border-slate-300 focus:border-brand-500"
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
      <span className="text-sm text-slate-500">{labelStatus}</span>
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
        <h2 className="text-lg font-semibold text-slate-800">{title}</h2>
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
      {body && <div className="text-sm text-slate-600">{body}</div>}
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

export function Badge({ children, color = "bg-slate-100 text-slate-700", className = "" }: BadgeProps) {
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
          {i > 0 && <span className="text-slate-300">/</span>}
          {item.href ? (
            <Link
              href={item.href}
              className="text-slate-500 hover:text-slate-900 transition"
            >
              {item.label}
            </Link>
          ) : (
            <span className="text-slate-800 font-medium" aria-current="page">
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
          className={`absolute z-40 mt-1 min-w-[12rem] rounded-md border border-slate-200 bg-white py-1 shadow-lg ${
            align === "right" ? "right-0" : "left-0"
          }`}
        >
          {children({ close })}
        </div>
      )}
    </div>
  );
}
