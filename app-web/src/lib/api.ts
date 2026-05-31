export interface ApiErrorDetail {
  field: string;
  message: string;
}

export class ApiError extends Error {
  readonly code: string;
  readonly details?: ApiErrorDetail[];

  constructor(code: string, message: string, details?: ApiErrorDetail[]) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.details = details;
  }
}

function getBaseUrl(): string {
  const url = process.env.NEXT_PUBLIC_API_URL;
  if (!url) {
    throw new Error("NEXT_PUBLIC_API_URL is not set");
  }
  return url;
}

/**
 * Parse a response body safely — returns null when the response is empty
 * (204 No Content or zero Content-Length).
 */
async function parseBody(res: Response): Promise<unknown> {
  // 204 No Content
  if (res.status === 204) return null;
  const text = await res.text();
  if (!text || text.trim() === "") return null;
  return JSON.parse(text);
}

export interface ApiRequestOptions {
  workspaceSlug?: string;
}

export async function apiGet<T>(path: string, opts?: ApiRequestOptions): Promise<T> {
  const base = getBaseUrl();
  const headers: Record<string, string> = {};
  if (opts?.workspaceSlug) {
    headers["X-Workspace-Slug"] = opts.workspaceSlug;
  }
  const res = await fetch(`${base}${path}`, { credentials: "include", headers });
  const body = await parseBody(res);

  if (!res.ok) {
    const err = (body as { error?: { code?: string; message?: string; details?: ApiErrorDetail[] } })?.error ?? {};
    throw new ApiError(
      err.code ?? "UNKNOWN_ERROR",
      err.message ?? `Request failed with status ${res.status}`,
      err.details
    );
  }

  return (body as { data: T }).data as T;
}

export async function apiPost<T>(path: string, reqBody: unknown, opts?: ApiRequestOptions): Promise<T | null> {
  const base = getBaseUrl();
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (opts?.workspaceSlug) {
    headers["X-Workspace-Slug"] = opts.workspaceSlug;
  }
  const res = await fetch(`${base}${path}`, {
    method: "POST",
    credentials: "include",
    headers,
    body: JSON.stringify(reqBody),
  });

  const body = await parseBody(res);

  if (!res.ok) {
    const err = (body as { error?: { code?: string; message?: string; details?: ApiErrorDetail[] } })?.error ?? {};
    throw new ApiError(
      err.code ?? "UNKNOWN_ERROR",
      err.message ?? `Request failed with status ${res.status}`,
      err.details
    );
  }

  if (body === null) return null;
  return (body as { data: T }).data as T;
}

export async function apiPut<T>(path: string, reqBody: unknown, opts?: ApiRequestOptions): Promise<T | null> {
  const base = getBaseUrl();
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (opts?.workspaceSlug) {
    headers["X-Workspace-Slug"] = opts.workspaceSlug;
  }
  const res = await fetch(`${base}${path}`, {
    method: "PUT",
    credentials: "include",
    headers,
    body: JSON.stringify(reqBody),
  });

  const body = await parseBody(res);

  if (!res.ok) {
    const err = (body as { error?: { code?: string; message?: string; details?: ApiErrorDetail[] } })?.error ?? {};
    throw new ApiError(
      err.code ?? "UNKNOWN_ERROR",
      err.message ?? `Request failed with status ${res.status}`,
      err.details
    );
  }

  if (body === null) return null;
  return (body as { data: T }).data as T;
}

export async function apiDelete(path: string, opts?: ApiRequestOptions): Promise<void> {
  const base = getBaseUrl();
  const headers: Record<string, string> = {};
  if (opts?.workspaceSlug) {
    headers["X-Workspace-Slug"] = opts.workspaceSlug;
  }
  const res = await fetch(`${base}${path}`, {
    method: "DELETE",
    credentials: "include",
    headers,
  });

  // parseBody handles 204 No Content AND an empty 200 body (returns null) —
  // never call res.json() directly: an empty body would throw SyntaxError.
  const body = await parseBody(res);

  if (!res.ok) {
    const err = (body as { error?: { code?: string; message?: string; details?: ApiErrorDetail[] } })?.error ?? {};
    throw new ApiError(
      err.code ?? "UNKNOWN_ERROR",
      err.message ?? `Request failed with status ${res.status}`,
      err.details
    );
  }

  // success: ignore any parsed body (DELETE returns void)
}
