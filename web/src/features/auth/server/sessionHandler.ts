// Server side of the web session: the /api/session/* route handlers (docs/AUTHENTICATION.md).
//
// The browser never sees the refresh token. These handlers call the Go API, move the refresh
// token into an HttpOnly cookie and return only the access token and user to the page. Mobile
// apps do not use this; they call /api/v1/auth/* directly.
//
// CSRF: every request must carry X-Session-Request: 1 (a custom header cross-site forms cannot
// send) and, when present, an Origin equal to this site. The refresh cookie is SameSite=Strict
// and scoped to /api/session.

import { API_VERSION } from "@/lib/env";
import type { ApiEnvelope } from "@/types/api";

import { REFRESH_COOKIE, clearedSessionCookies, sessionCookies, type CookieSpec } from "./cookies";

export const SESSION_HEADER = "X-Session-Request";

/** Actions under /api/session/<action> and the Go endpoint each one calls. */
const FORWARDED: Record<string, string> = {
  "sign-in": "/auth/sign-in",
  "sign-up/verify": "/auth/sign-up/verify",
  google: "/auth/google",
};

interface GoSession {
  access_token: string;
  access_token_expires_at: string;
  refresh_token: string;
  refresh_token_expires_at: string;
  remember: boolean;
  user: unknown;
  is_new_user: boolean;
}

export interface SessionHandlerConfig {
  apiOrigin: string;
  secureCookies: boolean;
  fetch?: typeof fetch;
}

function json(status: number, body: unknown, cookies: CookieSpec[] = []): Response {
  const headers = new Headers({ "Content-Type": "application/json", "Cache-Control": "no-store" });
  for (const c of cookies) headers.append("Set-Cookie", serializeCookie(c));
  return new Response(status === 204 ? null : JSON.stringify(body), { status, headers });
}

export function serializeCookie(c: CookieSpec): string {
  const parts = [`${c.name}=${encodeURIComponent(c.value)}`, `Path=${c.path}`, "HttpOnly"];
  parts.push(`SameSite=${c.sameSite === "strict" ? "Strict" : "Lax"}`);
  if (c.secure) parts.push("Secure");
  if (c.maxAge !== undefined) parts.push(`Max-Age=${c.maxAge}`);
  return parts.join("; ");
}

function failure(status: number, code: string, message: string, cookies: CookieSpec[] = []) {
  return json(status, { success: false, error: { code, message } }, cookies);
}

function readCookie(request: Request, name: string): string | undefined {
  const header = request.headers.get("cookie") ?? "";
  for (const part of header.split(";")) {
    const [key, ...rest] = part.trim().split("=");
    if (key === name) return decodeURIComponent(rest.join("="));
  }
  return undefined;
}

/** Rejects requests that did not come from this site's own JavaScript. */
function csrfError(request: Request): Response | null {
  if (request.headers.get(SESSION_HEADER) !== "1") {
    return failure(403, "FORBIDDEN", "Missing session request header.");
  }
  const origin = request.headers.get("origin");
  if (origin && origin !== new URL(request.url).origin) {
    return failure(403, "FORBIDDEN", "Cross-site session requests are not allowed.");
  }
  return null;
}

export function createSessionHandler(config: SessionHandlerConfig) {
  const doFetch = config.fetch ?? fetch;

  async function callApi(request: Request, path: string, body: unknown) {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      Accept: "application/json",
      "X-Client-Platform": "web",
    };
    // The Go API rate-limits per client IP; pass the browser's address along. The API honours it
    // only from TRUSTED_PROXIES (docs/AUTHENTICATION.md → Web).
    const forwarded = request.headers.get("x-forwarded-for") ?? request.headers.get("x-real-ip");
    if (forwarded) headers["X-Forwarded-For"] = forwarded;
    const ua = request.headers.get("user-agent");
    if (ua) headers["User-Agent"] = ua;

    const response = await doFetch(`${config.apiOrigin}/api/${API_VERSION}${path}`, {
      method: "POST",
      headers,
      body: JSON.stringify(body),
      cache: "no-store",
    });
    let envelope: ApiEnvelope<unknown> | undefined;
    if (response.status !== 204) {
      try {
        envelope = (await response.json()) as ApiEnvelope<unknown>;
      } catch {
        envelope = undefined;
      }
    }
    return { status: response.status, envelope };
  }

  /** Strips the refresh token from a session and stores it in cookies instead. */
  function established(data: GoSession, extraCookies: CookieSpec[] = []): Response {
    const cookies = sessionCookies({
      refreshToken: data.refresh_token,
      expiresAt: data.refresh_token_expires_at,
      remember: data.remember,
      secure: config.secureCookies,
    });
    return json(
      200,
      {
        success: true,
        data: {
          access_token: data.access_token,
          access_token_expires_at: data.access_token_expires_at,
          user: data.user,
          is_new_user: data.is_new_user,
        },
      },
      [...extraCookies, ...cookies],
    );
  }

  return async function handle(request: Request, action: string): Promise<Response> {
    const blocked = csrfError(request);
    if (blocked) return blocked;

    try {
      if (action === "refresh") {
        const token = readCookie(request, REFRESH_COOKIE);
        if (!token) {
          return failure(
            401,
            "UNAUTHORIZED",
            "Sign in to continue.",
            clearedSessionCookies(config.secureCookies),
          );
        }
        const { status, envelope } = await callApi(request, "/auth/refresh", {
          refresh_token: token,
        });
        if (status === 200 && envelope?.success) return established(envelope.data as GoSession);
        if (status === 401) {
          return json(401, envelope ?? {}, clearedSessionCookies(config.secureCookies));
        }
        return json(status, envelope ?? {});
      }

      if (action === "logout") {
        const token = readCookie(request, REFRESH_COOKIE);
        if (token) {
          // Best effort: the local sign-out happens even if the API is unreachable.
          await callApi(request, "/auth/logout", { refresh_token: token }).catch(() => undefined);
        }
        return json(204, null, clearedSessionCookies(config.secureCookies));
      }

      const path = FORWARDED[action];
      if (!path) return failure(404, "NOT_FOUND", "Route not found");

      let body: unknown;
      try {
        body = await request.json();
      } catch {
        return failure(400, "BAD_REQUEST", "Request body must be valid JSON");
      }
      const { status, envelope } = await callApi(request, path, body);
      if (status === 200 && envelope?.success) return established(envelope.data as GoSession);
      if (!envelope)
        return failure(502, "SERVICE_UNAVAILABLE", "Sign-in is not available right now.");
      return json(status, envelope);
    } catch {
      return failure(503, "SERVICE_UNAVAILABLE", "Sign-in is not available right now.");
    }
  };
}
