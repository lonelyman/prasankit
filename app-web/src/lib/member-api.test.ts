import { describe, it, expect, vi, beforeAll, afterAll } from "vitest";
import type { MockInstance } from "vitest";
import { listMembers, addMember, changeMemberRole, removeMember } from "./member-api";
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

const PROJECT_ID = "33333333-3333-3333-3333-333333333333";
const MEMBER_ID = "44444444-4444-4444-4444-444444444444";

function sampleMember(overrides: Record<string, unknown> = {}) {
  return {
    id: "44444444-4444-4444-4444-444444444444",
    workspace_id: "22222222-2222-2222-2222-222222222222",
    project_id: PROJECT_ID,
    workspace_membership_id: "55555555-5555-5555-5555-555555555555",
    project_role_code: "member",
    joined_at: "2026-05-31T00:00:00Z",
    removed_at: null,
    ...overrides,
  };
}

function lastCall(mockFetch: ReturnType<typeof vi.fn>): [string, RequestInit] {
  return (mockFetch as MockInstance).mock.calls[0] as [string, RequestInit];
}

describe("listMembers", () => {
  it("unwraps the {data:{items,count}} envelope", async () => {
    const items = [sampleMember({ display_name: "Alice" })];
    global.fetch = vi.fn().mockResolvedValueOnce(okJson({ data: { items, count: 1 } }));

    const result = await listMembers("my-ws", PROJECT_ID);

    expect(result.items).toEqual(items);
    expect(result.count).toBe(1);
  });

  it("returns empty items array without throwing", async () => {
    global.fetch = vi.fn().mockResolvedValueOnce(okJson({ data: { items: [], count: 0 } }));

    const result = await listMembers("my-ws", PROJECT_ID);

    expect(result.items).toEqual([]);
    expect(result.count).toBe(0);
  });

  it("sends X-Workspace-Slug header from slug arg", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce(okJson({ data: { items: [], count: 0 } }));
    global.fetch = mockFetch as unknown as typeof fetch;

    await listMembers("my-ws", PROJECT_ID);

    const headers = lastCall(mockFetch)[1].headers as Record<string, string>;
    expect(headers["X-Workspace-Slug"]).toBe("my-ws");
  });

  it("requests the project-scoped /members path", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce(okJson({ data: { items: [], count: 0 } }));
    global.fetch = mockFetch as unknown as typeof fetch;

    await listMembers("my-ws", PROJECT_ID);

    const url = lastCall(mockFetch)[0];
    expect(url).toContain(`/projects/${PROJECT_ID}/members`);
  });

  it("throws ApiError with code project.not_found on 404", async () => {
    global.fetch = vi
      .fn()
      .mockResolvedValueOnce(errJson(404, { error: { code: "project.not_found", message: "x" } }));

    await expect(listMembers("my-ws", PROJECT_ID)).rejects.toSatisfy(
      (err: unknown) => err instanceof ApiError && err.code === "project.not_found"
    );
  });
});

describe("addMember", () => {
  it("returns the created member from a 201 data envelope", async () => {
    const member = sampleMember();
    global.fetch = vi.fn().mockResolvedValueOnce(okJson({ data: member }, 201));

    const result = await addMember("my-ws", PROJECT_ID, {
      workspace_membership_id: "55555555-5555-5555-5555-555555555555",
      project_role_code: "member",
    });

    expect(result).toEqual(member);
  });

  it("sends Content-Type application/json with workspace_membership_id and project_role_code", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce(okJson({ data: sampleMember() }, 201));
    global.fetch = mockFetch as unknown as typeof fetch;

    await addMember("my-ws", PROJECT_ID, {
      workspace_membership_id: "55555555-5555-5555-5555-555555555555",
      project_role_code: "finance",
    });

    const init = lastCall(mockFetch)[1];
    const headers = init.headers as Record<string, string>;
    expect(headers["Content-Type"]).toBe("application/json");
    const body = JSON.parse(init.body as string);
    expect(body.workspace_membership_id).toBe("55555555-5555-5555-5555-555555555555");
    expect(body.project_role_code).toBe("finance");
  });

  it("uses the POST method to the /members path", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce(okJson({ data: sampleMember() }, 201));
    global.fetch = mockFetch as unknown as typeof fetch;

    await addMember("my-ws", PROJECT_ID, {
      workspace_membership_id: "55555555-5555-5555-5555-555555555555",
      project_role_code: "member",
    });

    const [url, init] = lastCall(mockFetch);
    expect(url).toContain(`/projects/${PROJECT_ID}/members`);
    expect(init.method).toBe("POST");
  });

  it("throws ApiError with code project_member.already_member on 409", async () => {
    global.fetch = vi
      .fn()
      .mockResolvedValueOnce(
        errJson(409, { error: { code: "project_member.already_member", message: "x" } })
      );

    await expect(
      addMember("my-ws", PROJECT_ID, {
        workspace_membership_id: "55555555-5555-5555-5555-555555555555",
        project_role_code: "member",
      })
    ).rejects.toSatisfy(
      (err: unknown) => err instanceof ApiError && err.code === "project_member.already_member"
    );
  });

  it("throws ApiError with code project_member.membership_not_eligible on 422", async () => {
    global.fetch = vi.fn().mockResolvedValueOnce(
      errJson(422, {
        error: {
          code: "project_member.membership_not_eligible",
          message: "x",
          details: { field: "workspace_membership_id" },
        },
      })
    );

    await expect(
      addMember("my-ws", PROJECT_ID, {
        workspace_membership_id: "55555555-5555-5555-5555-555555555555",
        project_role_code: "member",
      })
    ).rejects.toSatisfy(
      (err: unknown) =>
        err instanceof ApiError && err.code === "project_member.membership_not_eligible"
    );
  });

  it("throws ApiError with code project_member.invalid_role_code on 422", async () => {
    global.fetch = vi.fn().mockResolvedValueOnce(
      errJson(422, {
        error: {
          code: "project_member.invalid_role_code",
          message: "x",
          details: { field: "project_role_code" },
        },
      })
    );

    await expect(
      addMember("my-ws", PROJECT_ID, {
        workspace_membership_id: "55555555-5555-5555-5555-555555555555",
        project_role_code: "project_owner",
      })
    ).rejects.toSatisfy(
      (err: unknown) => err instanceof ApiError && err.code === "project_member.invalid_role_code"
    );
  });
});

describe("changeMemberRole", () => {
  it("puts new_role_code to the /members/:memberId/role endpoint", async () => {
    const mockFetch = vi
      .fn()
      .mockResolvedValueOnce(okJson({ data: sampleMember({ project_role_code: "finance" }) }));
    global.fetch = mockFetch as unknown as typeof fetch;

    await changeMemberRole("my-ws", PROJECT_ID, MEMBER_ID, "finance");

    const [url, init] = lastCall(mockFetch);
    expect(url).toContain(`/projects/${PROJECT_ID}/members/${MEMBER_ID}/role`);
    expect(init.method).toBe("PUT");
    const body = JSON.parse(init.body as string);
    expect(body.new_role_code).toBe("finance");
  });

  it("returns the member from a 200 data envelope", async () => {
    const member = sampleMember({ project_role_code: "finance" });
    global.fetch = vi.fn().mockResolvedValueOnce(okJson({ data: member }));

    const result = await changeMemberRole("my-ws", PROJECT_ID, MEMBER_ID, "finance");

    expect(result).toEqual(member);
  });

  it("throws ApiError with code project_member.owner_immutable on 422", async () => {
    global.fetch = vi
      .fn()
      .mockResolvedValueOnce(
        errJson(422, { error: { code: "project_member.owner_immutable", message: "x" } })
      );

    await expect(changeMemberRole("my-ws", PROJECT_ID, MEMBER_ID, "member")).rejects.toSatisfy(
      (err: unknown) => err instanceof ApiError && err.code === "project_member.owner_immutable"
    );
  });

  it("throws ApiError with code project_member.invalid_role_code on 422", async () => {
    global.fetch = vi.fn().mockResolvedValueOnce(
      errJson(422, {
        error: {
          code: "project_member.invalid_role_code",
          message: "x",
          details: { field: "new_role_code" },
        },
      })
    );

    await expect(
      changeMemberRole("my-ws", PROJECT_ID, MEMBER_ID, "project_owner")
    ).rejects.toSatisfy(
      (err: unknown) => err instanceof ApiError && err.code === "project_member.invalid_role_code"
    );
  });
});

describe("removeMember", () => {
  it("resolves void on 204 No Content", async () => {
    global.fetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 204,
      text: async () => "",
    } as unknown as Response);

    await expect(removeMember("my-ws", PROJECT_ID, MEMBER_ID)).resolves.toBeUndefined();
  });

  it("uses the DELETE method to the /members/:memberId path", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 204,
      text: async () => "",
    } as unknown as Response);
    global.fetch = mockFetch as unknown as typeof fetch;

    await removeMember("my-ws", PROJECT_ID, MEMBER_ID);

    const [url, init] = lastCall(mockFetch);
    expect(url).toContain(`/projects/${PROJECT_ID}/members/${MEMBER_ID}`);
    expect(init.method).toBe("DELETE");
  });

  it("throws ApiError with code project_member.owner_immutable on 422", async () => {
    global.fetch = vi
      .fn()
      .mockResolvedValueOnce(
        errJson(422, { error: { code: "project_member.owner_immutable", message: "x" } })
      );

    await expect(removeMember("my-ws", PROJECT_ID, MEMBER_ID)).rejects.toSatisfy(
      (err: unknown) => err instanceof ApiError && err.code === "project_member.owner_immutable"
    );
  });

  it("throws ApiError with code project_member.not_found on 404", async () => {
    global.fetch = vi
      .fn()
      .mockResolvedValueOnce(
        errJson(404, { error: { code: "project_member.not_found", message: "x" } })
      );

    await expect(removeMember("my-ws", PROJECT_ID, MEMBER_ID)).rejects.toSatisfy(
      (err: unknown) => err instanceof ApiError && err.code === "project_member.not_found"
    );
  });
});

describe("listWorkspaceMembers", () => {
  it("unwraps the {data:{items,count}} envelope", async () => {
    const items = [
      {
        workspace_membership_id: "55555555-5555-5555-5555-555555555555",
        display_name: "Alice",
        org_role_code: "admin",
      },
    ];
    global.fetch = vi.fn().mockResolvedValueOnce(okJson({ data: { items, count: 1 } }));

    const { listWorkspaceMembers } = await import("./workspace-api");
    const result = await listWorkspaceMembers("my-ws");

    expect(result.items).toEqual(items);
    expect(result.count).toBe(1);
  });

  it("sends X-Workspace-Slug header", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce(okJson({ data: { items: [], count: 0 } }));
    global.fetch = mockFetch as unknown as typeof fetch;

    const { listWorkspaceMembers } = await import("./workspace-api");
    await listWorkspaceMembers("my-ws");

    const headers = lastCall(mockFetch)[1].headers as Record<string, string>;
    expect(headers["X-Workspace-Slug"]).toBe("my-ws");
  });

  it("requests /api/v1/workspaces/members", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce(okJson({ data: { items: [], count: 0 } }));
    global.fetch = mockFetch as unknown as typeof fetch;

    const { listWorkspaceMembers } = await import("./workspace-api");
    await listWorkspaceMembers("my-ws");

    const url = lastCall(mockFetch)[0];
    expect(url).toContain("/api/v1/workspaces/members");
  });
});
