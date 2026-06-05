import { describe, it, expect, vi, beforeAll, afterAll } from "vitest";
import type { MockInstance } from "vitest";
import {
  listPositions,
  createPosition,
  updatePosition,
  deprecatePosition,
} from "./position-api";
import { ApiError } from "./api";

beforeAll(() => {
  process.env.NEXT_PUBLIC_API_URL = "http://localhost:8080";
});

afterAll(() => {
  delete process.env.NEXT_PUBLIC_API_URL;
});

function okJson(body: unknown, status = 200) {
  return {
    ok: true,
    status,
    text: async () => JSON.stringify(body),
  } as unknown as Response;
}

function errJson(status: number, body: unknown) {
  return {
    ok: false,
    status,
    text: async () => JSON.stringify(body),
  } as unknown as Response;
}

function lastCall(mockFetch: ReturnType<typeof vi.fn>): [string, RequestInit] {
  return (mockFetch as MockInstance).mock.calls[0] as [string, RequestInit];
}

function samplePosition(overrides: Record<string, unknown> = {}) {
  return {
    id: "11111111-1111-1111-1111-111111111111",
    workspace_id: "22222222-2222-2222-2222-222222222222",
    code: "lead_dev",
    label_th: "หัวหน้านักพัฒนา",
    label_en: "Lead Developer",
    description: null,
    sort_order: 1,
    is_system: false,
    status: "active",
    created_at: "2026-06-01T00:00:00Z",
    updated_at: "2026-06-01T00:00:00Z",
    ...overrides,
  };
}

const PAGINATION = { page: 1, limit: 100, total: 1, total_pages: 1, has_more: false };

describe("listPositions", () => {
  it("unwraps the {data:{items,pagination}} envelope", async () => {
    const items = [samplePosition()];
    global.fetch = vi.fn().mockResolvedValueOnce(okJson({ data: { items, pagination: PAGINATION } }));

    const result = await listPositions("my-ws", "project");

    expect(result.items).toEqual(items);
    expect(result.pagination.has_more).toBe(false);
  });

  it("hits the /project-positions path for kind project", async () => {
    const mockFetch = vi
      .fn()
      .mockResolvedValueOnce(okJson({ data: { items: [], pagination: PAGINATION } }));
    global.fetch = mockFetch as unknown as typeof fetch;

    await listPositions("my-ws", "project");

    expect(lastCall(mockFetch)[0]).toContain("/api/v1/workspaces/project-positions");
  });

  it("hits the /company-positions path for kind company", async () => {
    const mockFetch = vi
      .fn()
      .mockResolvedValueOnce(okJson({ data: { items: [], pagination: PAGINATION } }));
    global.fetch = mockFetch as unknown as typeof fetch;

    await listPositions("my-ws", "company");

    expect(lastCall(mockFetch)[0]).toContain("/api/v1/workspaces/company-positions");
  });

  it("appends ?status=active when passed and sends the slug header", async () => {
    const mockFetch = vi
      .fn()
      .mockResolvedValueOnce(okJson({ data: { items: [], pagination: PAGINATION } }));
    global.fetch = mockFetch as unknown as typeof fetch;

    await listPositions("my-ws", "project", { status: "active" });

    const [url, init] = lastCall(mockFetch);
    expect(url).toContain("status=active");
    expect((init.headers as Record<string, string>)["X-Workspace-Slug"]).toBe("my-ws");
  });
});

describe("createPosition", () => {
  it("POSTs application/json with code/label_th/label_en/sort_order and returns the 201 item", async () => {
    const created = samplePosition();
    const mockFetch = vi.fn().mockResolvedValueOnce(okJson({ data: created }, 201));
    global.fetch = mockFetch as unknown as typeof fetch;

    const result = await createPosition("my-ws", "project", {
      code: "lead_dev",
      label_th: "หัวหน้า",
      label_en: "Lead",
      sort_order: 1,
    });

    const init = lastCall(mockFetch)[1];
    expect(init.method).toBe("POST");
    expect((init.headers as Record<string, string>)["Content-Type"]).toBe("application/json");
    const body = JSON.parse(init.body as string);
    expect(body.code).toBe("lead_dev");
    expect(body.label_th).toBe("หัวหน้า");
    expect(body.label_en).toBe("Lead");
    expect(body.sort_order).toBe(1);
    expect(result).toEqual(created);
  });

  it("omits description from the body when not provided", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce(okJson({ data: samplePosition() }, 201));
    global.fetch = mockFetch as unknown as typeof fetch;

    await createPosition("my-ws", "company", {
      code: "senior_engineer",
      label_th: "วิศวกรอาวุโส",
      label_en: "Senior Engineer",
      sort_order: 0,
    });

    const body = JSON.parse(lastCall(mockFetch)[1].body as string);
    expect("description" in body).toBe(false);
  });

  it("includes description when provided", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce(okJson({ data: samplePosition() }, 201));
    global.fetch = mockFetch as unknown as typeof fetch;

    await createPosition("my-ws", "project", {
      code: "lead_dev",
      label_th: "หัวหน้า",
      label_en: "Lead",
      description: "leads the build",
      sort_order: 0,
    });

    const body = JSON.parse(lastCall(mockFetch)[1].body as string);
    expect(body.description).toBe("leads the build");
  });

  it("throws ApiError code project_position.code_taken on 409", async () => {
    global.fetch = vi.fn().mockResolvedValueOnce(
      errJson(409, { error: { code: "project_position.code_taken", message: "x", details: { field: "code" } } })
    );

    await expect(
      createPosition("my-ws", "project", { code: "lead_dev", label_th: "x", label_en: "y", sort_order: 0 })
    ).rejects.toSatisfy(
      (err: unknown) => err instanceof ApiError && err.code === "project_position.code_taken"
    );
  });

  it("throws ApiError code company_position.code_taken on 409", async () => {
    global.fetch = vi.fn().mockResolvedValueOnce(
      errJson(409, { error: { code: "company_position.code_taken", message: "x" } })
    );

    await expect(
      createPosition("my-ws", "company", { code: "boss", label_th: "x", label_en: "y", sort_order: 0 })
    ).rejects.toSatisfy(
      (err: unknown) => err instanceof ApiError && err.code === "company_position.code_taken"
    );
  });
});

describe("updatePosition", () => {
  it("PUTs to /:code with status in the body and returns the 200 item", async () => {
    const updated = samplePosition({ label_en: "Lead Dev 2" });
    const mockFetch = vi.fn().mockResolvedValueOnce(okJson({ data: updated }));
    global.fetch = mockFetch as unknown as typeof fetch;

    const result = await updatePosition("my-ws", "project", "lead_dev", {
      code: "lead_dev",
      label_th: "หัวหน้า",
      label_en: "Lead Dev 2",
      sort_order: 1,
      status: "active",
    });

    const [url, init] = lastCall(mockFetch);
    expect(url).toContain("/api/v1/workspaces/project-positions/lead_dev");
    expect(init.method).toBe("PUT");
    expect(JSON.parse(init.body as string).status).toBe("active");
    expect(result).toEqual(updated);
  });

  it("throws ApiError code project_position.code_immutable on 422", async () => {
    global.fetch = vi.fn().mockResolvedValueOnce(
      errJson(422, { error: { code: "project_position.code_immutable", message: "x", details: { field: "code" } } })
    );

    await expect(
      updatePosition("my-ws", "project", "lead_dev", {
        code: "other",
        label_th: "x",
        label_en: "y",
        sort_order: 0,
        status: "active",
      })
    ).rejects.toSatisfy(
      (err: unknown) => err instanceof ApiError && err.code === "project_position.code_immutable"
    );
  });

  it("throws ApiError code project_position.system_immutable on 422", async () => {
    global.fetch = vi.fn().mockResolvedValueOnce(
      errJson(422, { error: { code: "project_position.system_immutable", message: "x" } })
    );

    await expect(
      updatePosition("my-ws", "project", "sys", {
        code: "sys",
        label_th: "x",
        label_en: "y",
        sort_order: 0,
        status: "active",
      })
    ).rejects.toSatisfy(
      (err: unknown) => err instanceof ApiError && err.code === "project_position.system_immutable"
    );
  });

  it("throws ApiError code company_position.invalid_status on 422", async () => {
    global.fetch = vi.fn().mockResolvedValueOnce(
      errJson(422, { error: { code: "company_position.invalid_status", message: "x", details: { field: "status" } } })
    );

    await expect(
      updatePosition("my-ws", "company", "x", {
        code: "x",
        label_th: "x",
        label_en: "y",
        sort_order: 0,
        status: "bogus",
      })
    ).rejects.toSatisfy(
      (err: unknown) => err instanceof ApiError && err.code === "company_position.invalid_status"
    );
  });
});

describe("deprecatePosition", () => {
  it("POSTs to /:code/deprecate and returns the 200 item", async () => {
    const dep = samplePosition({ status: "deprecated" });
    const mockFetch = vi.fn().mockResolvedValueOnce(okJson({ data: dep }));
    global.fetch = mockFetch as unknown as typeof fetch;

    const result = await deprecatePosition("my-ws", "project", "lead_dev");

    const [url, init] = lastCall(mockFetch);
    expect(url).toContain("/api/v1/workspaces/project-positions/lead_dev/deprecate");
    expect(init.method).toBe("POST");
    expect(result.status).toBe("deprecated");
  });
});
