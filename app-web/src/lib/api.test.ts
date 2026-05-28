import { describe, it, expect, vi, beforeAll, afterAll } from "vitest";
import { apiGet, ApiError } from "./api";

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
      json: async () => ({ data: mockData }),
    } as Response);

    const result = await apiGet<typeof mockData>("/api/v1/health/ready");

    expect(result).toEqual(mockData);
  });

  it("throws ApiError with correct code on 503 error response", async () => {
    global.fetch = vi.fn().mockResolvedValueOnce({
      ok: false,
      status: 503,
      json: async () => ({
        error: {
          code: "SERVICE_UNAVAILABLE",
          message: "x",
        },
      }),
    } as Response);

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
