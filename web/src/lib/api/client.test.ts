import { describe, expect, it, vi } from "vitest";

import { ApiError, buildUrl, createApiClient } from "./client";

function jsonResponse(status: number, body: unknown, headers: Record<string, string> = {}) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json", ...headers },
  });
}

function clientWith(response: Response | Error) {
  const fetchMock = vi.fn(async () => {
    if (response instanceof Error) throw response;
    return response;
  });
  return { fetchMock, client: createApiClient({ baseUrl: "http://api.test", fetch: fetchMock }) };
}

describe("buildUrl", () => {
  it("versions the path and encodes query parameters", () => {
    expect(
      buildUrl("http://api.test", "v1", "/questions", {
        page: 2,
        skill: ["math", "rw"],
        q: undefined,
      }),
    ).toBe("http://api.test/api/v1/questions?page=2&skill=math&skill=rw");
  });
});

describe("createApiClient", () => {
  it("unwraps data and sends platform and auth headers", async () => {
    const { client, fetchMock } = clientWith(
      jsonResponse(200, { success: true, data: { id: "1" } }),
    );
    await expect(client.request("/me", { accessToken: "tok" })).resolves.toEqual({ id: "1" });

    const [, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit];
    const headers = init.headers as Record<string, string>;
    expect(headers["X-Client-Platform"]).toBe("web");
    expect(headers.Authorization).toBe("Bearer tok");
  });

  it("returns items and meta for paginated lists", async () => {
    const meta = { page: 1, page_size: 20, total: 1 };
    const { client } = clientWith(jsonResponse(200, { success: true, data: [{ id: "q1" }], meta }));
    await expect(client.page("/questions")).resolves.toEqual({ items: [{ id: "q1" }], meta });
  });

  it("throws ApiError with field errors for validation failures", async () => {
    const { client } = clientWith(
      jsonResponse(422, {
        success: false,
        error: {
          code: "VALIDATION_ERROR",
          message: "Some fields are invalid",
          details: { fields: { email: "must be a valid email address" } },
          request_id: "req-12345678",
        },
      }),
    );
    const err = await client.request("/x", { method: "POST", body: {} }).catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect(err).toMatchObject({
      status: 422,
      code: "VALIDATION_ERROR",
      fieldErrors: { email: "must be a valid email address" },
      requestId: "req-12345678",
    });
  });

  it("maps network failures and non-JSON bodies to ApiError", async () => {
    const offline = clientWith(new TypeError("fetch failed")).client;
    await expect(offline.request("/x")).rejects.toMatchObject({ code: "NETWORK_ERROR", status: 0 });

    const html = clientWith(new Response("<html>bad gateway</html>", { status: 502 })).client;
    await expect(html.request("/x")).rejects.toMatchObject({
      code: "INVALID_RESPONSE",
      status: 502,
    });
  });

  it("handles 204 No Content", async () => {
    const { client } = clientWith(new Response(null, { status: 204 }));
    await expect(client.request("/x", { method: "DELETE" })).resolves.toBeUndefined();
  });
});
