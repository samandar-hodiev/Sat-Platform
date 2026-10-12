import { describe, expect, it, vi } from "vitest";

import { createSessionHandler } from "./sessionHandler";

const goSession = {
  access_token: "access-1",
  access_token_expires_at: "2026-10-12T09:15:00Z",
  refresh_token: "refresh-secret",
  refresh_token_expires_at: new Date(Date.now() + 30 * 86400_000).toISOString(),
  remember: true,
  token_type: "Bearer",
  user: { id: "u1", email: "a@example.com", role: "learner" },
  is_new_user: true,
};

function ok(data: unknown, status = 200) {
  return new Response(JSON.stringify({ success: true, data }), { status });
}
function fail(status: number, code: string, message: string) {
  return new Response(JSON.stringify({ success: false, error: { code, message } }), { status });
}

function setup(apiResponse: () => Response | Promise<Response>) {
  const fetchMock = vi.fn<typeof fetch>(async () => apiResponse());
  const handle = createSessionHandler({
    apiOrigin: "http://api.test",
    secureCookies: true,
    fetch: fetchMock,
  });
  return { handle, fetchMock };
}

function request(
  action: string,
  init: { body?: unknown; cookie?: string; headers?: Record<string, string> } = {},
) {
  const headers = new Headers({
    "X-Session-Request": "1",
    "Content-Type": "application/json",
    ...init.headers,
  });
  if (init.cookie) headers.set("cookie", init.cookie);
  return new Request(`http://web.test/api/session/${action}`, {
    method: "POST",
    headers,
    body: init.body === undefined ? undefined : JSON.stringify(init.body),
  });
}

const cookies = (res: Response) => res.headers.getSetCookie();

describe("session handler", () => {
  it("signs in: refresh token goes to an HttpOnly cookie, never to the page", async () => {
    const { handle, fetchMock } = setup(() => ok(goSession));
    const res = await handle(
      request("sign-in", {
        body: { email: "a@example.com", password: "pw", remember: true },
        headers: { "x-forwarded-for": "203.0.113.9", "user-agent": "Test" },
      }),
      "sign-in",
    );
    expect(res.status).toBe(200);
    const text = await res.text();
    expect(text).not.toContain("refresh-secret");
    expect(JSON.parse(text).data).toEqual({
      access_token: "access-1",
      access_token_expires_at: goSession.access_token_expires_at,
      user: goSession.user,
      is_new_user: true,
    });

    const [refresh, marker] = cookies(res);
    expect(refresh).toMatch(
      /^sat_refresh=refresh-secret; Path=\/api\/session; HttpOnly; SameSite=Strict; Secure; Max-Age=\d+$/,
    );
    expect(marker).toMatch(
      /^sat_signed_in=1; Path=\/; HttpOnly; SameSite=Lax; Secure; Max-Age=\d+$/,
    );

    const [url, init] = fetchMock.mock.calls[0]!;
    expect(url).toBe("http://api.test/api/v1/auth/sign-in");
    const headers = init?.headers as Record<string, string>;
    expect(headers["X-Forwarded-For"]).toBe("203.0.113.9");
    expect(headers["X-Client-Platform"]).toBe("web");
  });

  it("uses browser-session cookies without Remember me", async () => {
    const { handle } = setup(() => ok({ ...goSession, remember: false }));
    const res = await handle(request("sign-in", { body: {} }), "sign-in");
    for (const c of cookies(res)) expect(c).not.toContain("Max-Age");
  });

  it("forwards sign-up verify and google to their API routes", async () => {
    const { handle, fetchMock } = setup(() => ok(goSession));
    await handle(
      request("sign-up/verify", { body: { email: "a@example.com", code: "123456" } }),
      "sign-up/verify",
    );
    await handle(request("google", { body: { id_token: "t" } }), "google");
    expect(fetchMock.mock.calls.map((c) => c[0])).toEqual([
      "http://api.test/api/v1/auth/sign-up/verify",
      "http://api.test/api/v1/auth/google",
    ]);
  });

  it("passes API errors through unchanged", async () => {
    const { handle } = setup(() => fail(401, "UNAUTHORIZED", "Email or password is incorrect."));
    const res = await handle(request("sign-in", { body: {} }), "sign-in");
    expect(res.status).toBe(401);
    expect((await res.json()).error.message).toBe("Email or password is incorrect.");
    expect(cookies(res)).toEqual([]);
  });

  it("rejects requests without the session header or from another origin", async () => {
    const { handle, fetchMock } = setup(() => ok(goSession));
    const noHeader = new Request("http://web.test/api/session/sign-in", {
      method: "POST",
      body: "{}",
    });
    expect((await handle(noHeader, "sign-in")).status).toBe(403);
    const crossSite = request("sign-in", { body: {}, headers: { origin: "https://evil.example" } });
    expect((await handle(crossSite, "sign-in")).status).toBe(403);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("refreshes with the cookie and rotates it", async () => {
    const { handle, fetchMock } = setup(() => ok({ ...goSession, refresh_token: "refresh-2" }));
    const res = await handle(
      request("refresh", { cookie: "sat_refresh=refresh-1; other=x" }),
      "refresh",
    );
    expect(res.status).toBe(200);
    expect(JSON.parse(fetchMock.mock.calls[0]![1]!.body as string)).toEqual({
      refresh_token: "refresh-1",
    });
    expect(cookies(res)[0]).toContain("sat_refresh=refresh-2");
  });

  it("clears cookies when the session has ended", async () => {
    const { handle } = setup(() => fail(401, "UNAUTHORIZED", "Your session has ended."));
    const res = await handle(request("refresh", { cookie: "sat_refresh=old" }), "refresh");
    expect(res.status).toBe(401);
    expect(cookies(res).every((c) => c.includes("Max-Age=0"))).toBe(true);

    const none = await setup(() => ok(goSession)).handle(request("refresh"), "refresh");
    expect(none.status).toBe(401);
  });

  it("keeps cookies when the API is down during refresh", async () => {
    const { handle } = setup(() => {
      throw new TypeError("fetch failed");
    });
    const res = await handle(request("refresh", { cookie: "sat_refresh=r" }), "refresh");
    expect(res.status).toBe(503);
    expect(cookies(res)).toEqual([]);
  });

  it("logs out on the API and clears cookies even if the API fails", async () => {
    const { handle, fetchMock } = setup(() => {
      throw new TypeError("down");
    });
    const res = await handle(request("logout", { cookie: "sat_refresh=r" }), "logout");
    expect(res.status).toBe(204);
    expect(fetchMock).toHaveBeenCalledOnce();
    expect(cookies(res)).toHaveLength(2);
  });

  it("404s unknown actions and 400s bad JSON", async () => {
    const { handle } = setup(() => ok(goSession));
    expect((await handle(request("password/reset", { body: {} }), "password/reset")).status).toBe(
      404,
    );
    const bad = new Request("http://web.test/api/session/sign-in", {
      method: "POST",
      headers: { "X-Session-Request": "1" },
      body: "{",
    });
    expect((await handle(bad, "sign-in")).status).toBe(400);
  });
});
