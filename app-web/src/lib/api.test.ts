import { describe, it, expect, vi, beforeAll, afterAll } from "vitest";
import { apiGet, apiPost, ApiError } from "./api";
import type { MockInstance } from "vitest";

beforeAll(() => {
  process.env.NEXT_PUBLIC_API_URL = "http://localhost:8080";
});

afterAll(() => {
  delete process.env.NEXT_PUBLIC_API_URL;
});

describe("apiGet", () => {
  it("returns data on 200 success response", async () => {
    const mockData = { status: "ok" };

    global.fetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 200,
      text: async () => JSON.stringify({ data: mockData }),
    } as unknown as Response);

    const result = await apiGet<typeof mockData>("/api/v1/health/ready");

    expect(result).toEqual(mockData);
  });

  it("throws ApiError with correct code on 503 error response", async () => {
    global.fetch = vi.fn().mockResolvedValueOnce({
      ok: false,
      status: 503,
      text: async () =>
        JSON.stringify({
          error: {
            code: "SERVICE_UNAVAILABLE",
            message: "x",
          },
        }),
    } as unknown as Response);

    await expect(apiGet("/api/v1/health/ready")).rejects.toSatisfy(
      (err: unknown) =>
        err instanceof ApiError && err.code === "SERVICE_UNAVAILABLE"
    );
  });

  it("throws when NEXT_PUBLIC_API_URL is not set", async () => {
    const saved = process.env.NEXT_PUBLIC_API_URL;
    delete process.env.NEXT_PUBLIC_API_URL;

    await expect(apiGet("/anything")).rejects.toThrow(
      "NEXT_PUBLIC_API_URL is not set"
    );

    process.env.NEXT_PUBLIC_API_URL = saved;
  });
});

describe("apiPost", () => {
  it("returns data on 200 JSON response", async () => {
    const mockData = { id: "abc", primary_email: "a@b.com" };

    global.fetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 200,
      text: async () => JSON.stringify({ data: mockData }),
    } as unknown as Response);

    const result = await apiPost<typeof mockData>("/api/v1/auth/login", {
      email: "a@b.com",
      password: "password1",
    });

    expect(result).toEqual(mockData);
  });

  it("returns null on 204 No Content", async () => {
    global.fetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 204,
      text: async () => "",
    } as unknown as Response);

    const result = await apiPost("/api/v1/auth/logout", {});

    expect(result).toBeNull();
  });

  it("throws ApiError with correct code on error envelope", async () => {
    global.fetch = vi.fn().mockResolvedValueOnce({
      ok: false,
      status: 401,
      text: async () =>
        JSON.stringify({
          error: {
            code: "auth.invalid_credentials",
            message: "Invalid email or password",
          },
        }),
    } as unknown as Response);

    await expect(
      apiPost("/api/v1/auth/login", { email: "x@y.com", password: "wrong" })
    ).rejects.toSatisfy(
      (err: unknown) =>
        err instanceof ApiError && err.code === "auth.invalid_credentials"
    );
  });

  it("preserves details array when present in error envelope", async () => {
    const details = [
      { field: "email", message: "Invalid email format" },
      { field: "password", message: "Password must be at least 8 characters" },
    ];

    global.fetch = vi.fn().mockResolvedValueOnce({
      ok: false,
      status: 400,
      text: async () =>
        JSON.stringify({
          error: {
            code: "validation.invalid_input",
            message: "Validation failed",
            details,
          },
        }),
    } as unknown as Response);

    let caught: unknown;
    try {
      await apiPost("/api/v1/auth/signup", {});
    } catch (err) {
      caught = err;
    }

    expect(caught).toBeInstanceOf(ApiError);
    expect((caught as ApiError).code).toBe("validation.invalid_input");
    expect((caught as ApiError).details).toEqual(details);
  });
});

describe("apiGet — workspaceSlug header", () => {
  it("sends X-Workspace-Slug header when workspaceSlug option is provided", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 200,
      text: async () => JSON.stringify({ data: { id: "ws-1" } }),
    } as unknown as Response);
    global.fetch = mockFetch as unknown as typeof fetch;

    await apiGet("/api/v1/workspaces/current", { workspaceSlug: "my-ws" });

    const callArgs = (mockFetch as MockInstance).mock.calls[0] as [string, RequestInit];
    const headers = callArgs[1].headers as Record<string, string>;
    expect(headers["X-Workspace-Slug"]).toBe("my-ws");
  });

  it("does NOT send X-Workspace-Slug header when workspaceSlug is omitted", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 200,
      text: async () => JSON.stringify({ data: { status: "ok" } }),
    } as unknown as Response);
    global.fetch = mockFetch as unknown as typeof fetch;

    await apiGet("/api/v1/health/ready");

    const callArgs = (mockFetch as MockInstance).mock.calls[0] as [string, RequestInit];
    const headers = (callArgs[1].headers ?? {}) as Record<string, string>;
    expect(headers["X-Workspace-Slug"]).toBeUndefined();
  });
});

describe("apiPost — workspaceSlug header", () => {
  it("sends X-Workspace-Slug header when workspaceSlug option is provided", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 201,
      text: async () => JSON.stringify({ data: { id: "inv-1" } }),
    } as unknown as Response);
    global.fetch = mockFetch as unknown as typeof fetch;

    await apiPost("/api/v1/workspaces/invitations", { email: "a@b.com", org_role_code: "user" }, { workspaceSlug: "my-ws" });

    const callArgs = (mockFetch as MockInstance).mock.calls[0] as [string, RequestInit];
    const headers = callArgs[1].headers as Record<string, string>;
    expect(headers["X-Workspace-Slug"]).toBe("my-ws");
  });

  it("does NOT send X-Workspace-Slug header when workspaceSlug is omitted", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 204,
      text: async () => "",
    } as unknown as Response);
    global.fetch = mockFetch as unknown as typeof fetch;

    await apiPost("/api/v1/auth/logout", {});

    const callArgs = (mockFetch as MockInstance).mock.calls[0] as [string, RequestInit];
    const headers = (callArgs[1].headers ?? {}) as Record<string, string>;
    expect(headers["X-Workspace-Slug"]).toBeUndefined();
  });
});
