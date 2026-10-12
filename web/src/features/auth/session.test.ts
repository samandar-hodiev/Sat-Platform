// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "@/lib/api/client";

import {
  getAccessToken,
  getSessionState,
  refreshSession,
  resetSessionForTests,
  signOut,
  withAuth,
} from "./session";

const user = {
  id: "u1",
  email: "a@example.com",
  role: "learner",
  email_verified: true,
  has_password: false,
  created_at: "2026-10-12T00:00:00Z",
};

function sessionBody(token: string, expiresInMs = 15 * 60_000) {
  return {
    success: true,
    data: {
      access_token: token,
      access_token_expires_at: new Date(Date.now() + expiresInMs).toISOString(),
      user,
      is_new_user: false,
    },
  };
}

const respond = (status: number, body?: unknown) =>
  new Response(body === undefined ? null : JSON.stringify(body), { status });

let fetchMock: ReturnType<typeof vi.fn>;

beforeEach(() => {
  resetSessionForTests();
  fetchMock = vi.fn();
  vi.stubGlobal("fetch", fetchMock);
});
afterEach(() => vi.unstubAllGlobals());

describe("session", () => {
  it("refresh is single-flight and sends the CSRF header", async () => {
    fetchMock.mockResolvedValue(respond(200, sessionBody("t1")));
    const results = await Promise.all([refreshSession(), refreshSession(), getAccessToken()]);
    expect(results).toEqual([true, true, "t1"]);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0]!;
    expect(url).toBe("/api/session/refresh");
    expect((init as RequestInit).headers).toMatchObject({ "X-Session-Request": "1" });
    expect(getSessionState()).toEqual({ status: "signed-in", user });
  });

  it("reuses a fresh token and refreshes one about to expire", async () => {
    fetchMock.mockResolvedValueOnce(respond(200, sessionBody("t1", 10_000)));
    fetchMock.mockResolvedValueOnce(respond(200, sessionBody("t2")));
    await refreshSession();
    expect(await getAccessToken()).toBe("t2"); // t1 is within the 30s margin
    expect(await getAccessToken()).toBe("t2");
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("signs out locally when the refresh is rejected", async () => {
    fetchMock.mockResolvedValue(
      respond(401, { success: false, error: { code: "UNAUTHORIZED", message: "x" } }),
    );
    expect(await refreshSession()).toBe(false);
    expect(getSessionState()).toEqual({ status: "signed-out" });
    expect(await getAccessToken()).toBeNull();
  });

  it("keeps the state and throws on network failure", async () => {
    fetchMock.mockRejectedValue(new TypeError("offline"));
    await expect(refreshSession()).rejects.toMatchObject({ code: "NETWORK_ERROR" });
    expect(getSessionState()).toEqual({ status: "loading" });
  });

  it("withAuth retries once after a 401", async () => {
    fetchMock.mockResolvedValueOnce(respond(200, sessionBody("old")));
    fetchMock.mockResolvedValueOnce(respond(200, sessionBody("new")));
    const call = vi.fn(async (token: string) => {
      if (token === "old") throw new ApiError(401, { code: "UNAUTHORIZED", message: "expired" });
      return "data";
    });
    await expect(withAuth(call)).resolves.toBe("data");
    expect(call.mock.calls.map((c) => c[0])).toEqual(["old", "new"]);
  });

  it("withAuth does not retry other errors", async () => {
    fetchMock.mockResolvedValueOnce(respond(200, sessionBody("t")));
    const call = vi.fn(async () => {
      throw new ApiError(403, { code: "FORBIDDEN", message: "no" });
    });
    await expect(withAuth(call)).rejects.toMatchObject({ code: "FORBIDDEN" });
    expect(call).toHaveBeenCalledTimes(1);
  });

  it("sign out clears the session even if the request fails", async () => {
    fetchMock.mockResolvedValueOnce(respond(200, sessionBody("t")));
    await refreshSession();
    fetchMock.mockRejectedValueOnce(new TypeError("offline"));
    await expect(signOut()).rejects.toBeInstanceOf(ApiError);
    expect(getSessionState()).toEqual({ status: "signed-out" });
  });
});
