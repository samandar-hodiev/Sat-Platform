// Cookie names and options of the web session (docs/AUTHENTICATION.md → Web).
// Shared by the session route handlers and proxy.ts; contains no secrets.

/** Refresh token. HttpOnly, sent only to /api/session/*, never readable by page JavaScript. */
export const REFRESH_COOKIE = "sat_refresh";

/**
 * "A session probably exists" marker for the route guard (proxy.ts). Holds no secret: the real
 * check is the refresh call. Path=/ so the guard sees it on every page.
 */
export const SIGNED_IN_COOKIE = "sat_signed_in";

export const REFRESH_COOKIE_PATH = "/api/session";

export interface CookieSpec {
  name: string;
  value: string;
  httpOnly: true;
  secure: boolean;
  sameSite: "strict" | "lax";
  path: string;
  /** Seconds. Absent: a browser-session cookie ("Remember me" off). */
  maxAge?: number;
}

export function sessionCookies(input: {
  refreshToken: string;
  expiresAt: string;
  remember: boolean;
  secure: boolean;
  now?: number;
}): CookieSpec[] {
  const seconds = Math.floor((Date.parse(input.expiresAt) - (input.now ?? Date.now())) / 1000);
  const maxAge = input.remember && seconds > 0 ? seconds : undefined;
  return [
    {
      name: REFRESH_COOKIE,
      value: input.refreshToken,
      httpOnly: true,
      secure: input.secure,
      sameSite: "strict",
      path: REFRESH_COOKIE_PATH,
      maxAge,
    },
    {
      name: SIGNED_IN_COOKIE,
      value: "1",
      httpOnly: true,
      secure: input.secure,
      // Lax, so following a link from an email into /app still passes the guard.
      sameSite: "lax",
      path: "/",
      maxAge,
    },
  ];
}

export function clearedSessionCookies(secure: boolean): CookieSpec[] {
  return [
    {
      name: REFRESH_COOKIE,
      value: "",
      httpOnly: true,
      secure,
      sameSite: "strict",
      path: REFRESH_COOKIE_PATH,
      maxAge: 0,
    },
    {
      name: SIGNED_IN_COOKIE,
      value: "",
      httpOnly: true,
      secure,
      sameSite: "lax",
      path: "/",
      maxAge: 0,
    },
  ];
}
