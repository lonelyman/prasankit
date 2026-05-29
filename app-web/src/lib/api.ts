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

export async function apiGet<T>(path: string): Promise<T> {
  const base = getBaseUrl();
  const res = await fetch(`${base}${path}`, { credentials: "include" });
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

export async function apiPost<T>(path: string, reqBody: unknown): Promise<T | null> {
  const base = getBaseUrl();
  const res = await fetch(`${base}${path}`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
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
