import { describe, it, expect, vi, beforeAll, afterAll } from "vitest";
import type { MockInstance } from "vitest";
import {
  listProjects,
  createProject,
  getProject,
  updateProject,
  deleteProject,
  changeProjectStatus,
} from "./project-api";
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

function sampleProject(overrides: Record<string, unknown> = {}) {
  return {
    id: "11111111-1111-1111-1111-111111111111",
    workspace_id: "22222222-2222-2222-2222-222222222222",
    project_name: "Demo",
    slug: null,
    project_type_code: "internal",
    project_status_code: "draft",
    requesting_unit: null,
    description: null,
    start_date: null,
    end_date: null,
    created_at: "2026-05-31T00:00:00Z",
    updated_at: "2026-05-31T00:00:00Z",
    ...overrides,
  };
}

function lastCall(mockFetch: ReturnType<typeof vi.fn>): [string, RequestInit] {
  return (mockFetch as MockInstance).mock.calls[0] as [string, RequestInit];
}

describe("listProjects", () => {
  it("unwraps the double-nested {data:{items,pagination}} envelope", async () => {
    const items = [sampleProject()];
    const pagination = { page: 1, limit: 10, total: 1, total_pages: 1, has_more: false };
    global.fetch = vi.fn().mockResolvedValueOnce(okJson({ data: { items, pagination } }));

    const result = await listProjects("my-ws");

    expect(result.items).toEqual(items);
    expect(result.pagination).toEqual(pagination);
  });

  it("returns empty items array without throwing", async () => {
    const pagination = { page: 1, limit: 10, total: 0, total_pages: 0, has_more: false };
    global.fetch = vi.fn().mockResolvedValueOnce(okJson({ data: { items: [], pagination } }));

    const result = await listProjects("my-ws");

    expect(result.items).toEqual([]);
  });

  it("appends page and limit query params when provided", async () => {
    const mockFetch = vi
      .fn()
      .mockResolvedValueOnce(okJson({ data: { items: [], pagination: {} } }));
    global.fetch = mockFetch as unknown as typeof fetch;

    await listProjects("my-ws", { page: 2, limit: 25 });

    const url = lastCall(mockFetch)[0];
    expect(url).toContain("page=2");
    expect(url).toContain("limit=25");
  });

  it("appends project_status_code and project_type_code filters when non-empty", async () => {
    const mockFetch = vi
      .fn()
      .mockResolvedValueOnce(okJson({ data: { items: [], pagination: {} } }));
    global.fetch = mockFetch as unknown as typeof fetch;

    await listProjects("my-ws", { project_status_code: "active", project_type_code: "client" });

    const url = lastCall(mockFetch)[0];
    expect(url).toContain("project_status_code=active");
    expect(url).toContain("project_type_code=client");
  });

  it("omits filter params when empty string", async () => {
    const mockFetch = vi
      .fn()
      .mockResolvedValueOnce(okJson({ data: { items: [], pagination: {} } }));
    global.fetch = mockFetch as unknown as typeof fetch;

    await listProjects("my-ws", { project_status_code: "", project_type_code: "" });

    const url = lastCall(mockFetch)[0];
    expect(url).not.toContain("project_status_code");
    expect(url).not.toContain("project_type_code");
  });

  it("sends X-Workspace-Slug header from slug arg", async () => {
    const mockFetch = vi
      .fn()
      .mockResolvedValueOnce(okJson({ data: { items: [], pagination: {} } }));
    global.fetch = mockFetch as unknown as typeof fetch;

    await listProjects("my-ws");

    const headers = lastCall(mockFetch)[1].headers as Record<string, string>;
    expect(headers["X-Workspace-Slug"]).toBe("my-ws");
  });

  it("surfaces validation.invalid_input as ApiError on bad page", async () => {
    global.fetch = vi
      .fn()
      .mockResolvedValueOnce(
        errJson(400, { error: { code: "validation.invalid_input", message: "bad page" } })
      );

    await expect(listProjects("my-ws", { page: 0 })).rejects.toSatisfy(
      (err: unknown) => err instanceof ApiError && err.code === "validation.invalid_input"
    );
  });
});

describe("createProject", () => {
  it("returns the created project from a 201 data envelope", async () => {
    const proj = sampleProject({ project_name: "New" });
    global.fetch = vi.fn().mockResolvedValueOnce(okJson({ data: proj }, 201));

    const result = await createProject("my-ws", {
      project_name: "New",
      project_type_code: "internal",
      project_status_code: "draft",
    });

    expect(result).toEqual(proj);
  });

  it("sends Content-Type application/json and the input body", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce(okJson({ data: sampleProject() }, 201));
    global.fetch = mockFetch as unknown as typeof fetch;

    await createProject("my-ws", {
      project_name: "New",
      project_type_code: "internal",
      project_status_code: "draft",
    });

    const init = lastCall(mockFetch)[1];
    const headers = init.headers as Record<string, string>;
    expect(headers["Content-Type"]).toBe("application/json");
    const body = JSON.parse(init.body as string);
    expect(body.project_name).toBe("New");
  });

  it("omits slug from the body when not provided", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce(okJson({ data: sampleProject() }, 201));
    global.fetch = mockFetch as unknown as typeof fetch;

    await createProject("my-ws", {
      project_name: "New",
      project_type_code: "internal",
      project_status_code: "draft",
    });

    const body = JSON.parse(lastCall(mockFetch)[1].body as string);
    expect("slug" in body).toBe(false);
  });

  it("throws ApiError with code project.slug_taken on 409", async () => {
    global.fetch = vi
      .fn()
      .mockResolvedValueOnce(errJson(409, { error: { code: "project.slug_taken", message: "x" } }));

    await expect(
      createProject("my-ws", {
        project_name: "New",
        project_type_code: "internal",
        project_status_code: "draft",
      })
    ).rejects.toSatisfy(
      (err: unknown) => err instanceof ApiError && err.code === "project.slug_taken"
    );
  });

  it("throws ApiError with code project.invalid_master_code on 422", async () => {
    global.fetch = vi.fn().mockResolvedValueOnce(
      errJson(422, {
        error: {
          code: "project.invalid_master_code",
          message: "x",
          details: { field: "project_status_code" },
        },
      })
    );

    await expect(
      createProject("my-ws", {
        project_name: "New",
        project_type_code: "internal",
        project_status_code: "bogus",
      })
    ).rejects.toSatisfy(
      (err: unknown) => err instanceof ApiError && err.code === "project.invalid_master_code"
    );
  });
});

describe("getProject", () => {
  it("returns the project from a 200 data envelope", async () => {
    const proj = sampleProject();
    global.fetch = vi.fn().mockResolvedValueOnce(okJson({ data: proj }));

    const result = await getProject("my-ws", proj.id);

    expect(result).toEqual(proj);
  });

  it("throws ApiError with code project.not_found on 404", async () => {
    global.fetch = vi
      .fn()
      .mockResolvedValueOnce(errJson(404, { error: { code: "project.not_found", message: "x" } }));

    await expect(getProject("my-ws", "missing")).rejects.toSatisfy(
      (err: unknown) => err instanceof ApiError && err.code === "project.not_found"
    );
  });
});

describe("updateProject", () => {
  it("returns the updated project from a 200 data envelope", async () => {
    const proj = sampleProject({ project_name: "Updated" });
    global.fetch = vi.fn().mockResolvedValueOnce(okJson({ data: proj }));

    const result = await updateProject("my-ws", proj.id, {
      project_name: "Updated",
      project_type_code: "internal",
    });

    expect(result).toEqual(proj);
  });

  it("does not send project_status_code in the request body", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce(okJson({ data: sampleProject() }));
    global.fetch = mockFetch as unknown as typeof fetch;

    await updateProject("my-ws", "id", {
      project_name: "Updated",
      project_type_code: "internal",
    });

    const body = JSON.parse(lastCall(mockFetch)[1].body as string);
    expect("project_status_code" in body).toBe(false);
  });

  it("uses the PUT method", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce(okJson({ data: sampleProject() }));
    global.fetch = mockFetch as unknown as typeof fetch;

    await updateProject("my-ws", "id", {
      project_name: "Updated",
      project_type_code: "internal",
    });

    expect(lastCall(mockFetch)[1].method).toBe("PUT");
  });
});

describe("deleteProject", () => {
  it("resolves void on 204 No Content", async () => {
    global.fetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 204,
      text: async () => "",
    } as unknown as Response);

    await expect(deleteProject("my-ws", "id")).resolves.toBeUndefined();
  });

  it("throws ApiError with code project.not_found on 404", async () => {
    global.fetch = vi
      .fn()
      .mockResolvedValueOnce(errJson(404, { error: { code: "project.not_found", message: "x" } }));

    await expect(deleteProject("my-ws", "id")).rejects.toSatisfy(
      (err: unknown) => err instanceof ApiError && err.code === "project.not_found"
    );
  });

  it("uses the DELETE method", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 204,
      text: async () => "",
    } as unknown as Response);
    global.fetch = mockFetch as unknown as typeof fetch;

    await deleteProject("my-ws", "id");

    expect(lastCall(mockFetch)[1].method).toBe("DELETE");
  });
});

describe("changeProjectStatus", () => {
  it("posts new_status_code to the /status endpoint", async () => {
    const mockFetch = vi
      .fn()
      .mockResolvedValueOnce(okJson({ data: sampleProject({ project_status_code: "active" }) }));
    global.fetch = mockFetch as unknown as typeof fetch;

    await changeProjectStatus("my-ws", "id", "active");

    const [url, init] = lastCall(mockFetch);
    expect(url).toContain("/status");
    expect(init.method).toBe("POST");
    const body = JSON.parse(init.body as string);
    expect(body.new_status_code).toBe("active");
  });

  it("returns the project with updated project_status_code", async () => {
    const proj = sampleProject({ project_status_code: "active" });
    global.fetch = vi.fn().mockResolvedValueOnce(okJson({ data: proj }));

    const result = await changeProjectStatus("my-ws", "id", "active");

    expect(result.project_status_code).toBe("active");
  });

  it("throws ApiError with code project.invalid_master_code on 422", async () => {
    global.fetch = vi.fn().mockResolvedValueOnce(
      errJson(422, {
        error: {
          code: "project.invalid_master_code",
          message: "x",
          details: { field: "project_status_code" },
        },
      })
    );

    await expect(changeProjectStatus("my-ws", "id", "bogus")).rejects.toSatisfy(
      (err: unknown) => err instanceof ApiError && err.code === "project.invalid_master_code"
    );
  });
});
