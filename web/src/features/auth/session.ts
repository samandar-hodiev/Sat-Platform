"use client";

// Browser session (session 1.01, docs/AUTHENTICATION.md → Web).
//
// - The access token lives in memory only (this module), never in storage.
// - The refresh token is an HttpOnly cookie handled by /api/session/*; this code never sees it.
// - refreshSession() is single-flight: concurrent callers share one request.
// - withAuth() gives feature code a valid access token and retries once after a refresh when the
//   API says the token expired.

import { useSyncExternalStore } from "react";

import { ApiError } from "@/lib/api/client";
import type { ApiEnvelope } from "@/types/api";

export interface SessionUser {
  id: string;
  email: string;
  role: "learner";
  email_verified: boolean;
  has_password: boolean;
  created_at: string;
}

export interface EstablishedSession {
  access_token: string;
  access_token_expires_at: string;
  user: SessionUser;
  is_new_user: boolean;
}

export type SessionState =
  { status: "loading" } | { status: "signed-out" } | { status: "signed-in"; user: SessionUser };

/** Refresh this long before the access token expires, so requests never race the expiry. */
const EXPIRY_MARGIN_MS = 30_000;

let state: SessionState = { status: "loading" };
let accessToken: string | null = null;
let accessExpiresAt = 0;
let inflight: Promise<boolean> | null = null;
const listeners = new Set<() => void>();

function setState(next: SessionState) {
  state = next;
  for (const l of listeners) l();
}

export function getSessionState(): SessionState {
  return state;
}

export function subscribeSession(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

const SERVER_STATE: SessionState = { status: "loading" };

export function useSession(): SessionState {
  return useSyncExternalStore(subscribeSession, getSessionState, () => SERVER_STATE);
}

/** POST to a same-origin /api/session/* route and unwrap the envelope like the API client. */
export async function sessionRequest<T>(action: string, body?: unknown): Promise<T> {
  let response: Response;
  try {
    response = await fetch(`/api/session/${action}`, {
      method: "POST",
      credentials: "same-origin",
      cache: "no-store",
      headers: {
        "X-Session-Request": "1",
        Accept: "application/json",
        ...(body === undefined ? {} : { "Content-Type": "application/json" }),
      },
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    throw new ApiError(0, {
      code: "NETWORK_ERROR",
      message: "Could not reach the server. Check your connection.",
    });
  }
  if (response.status === 204) return undefined as T;
  let envelope: ApiEnvelope<T>;
  try {
    envelope = (await response.json()) as ApiEnvelope<T>;
  } catch {
    throw new ApiError(response.status, {
      code: "INVALID_RESPONSE",
      message: "Unexpected response from the server.",
    });
  }
  if (!response.ok || !envelope.success) {
    const error = envelope.success ? undefined : envelope.error;
    throw new ApiError(response.status, {
      code: error?.code ?? "INVALID_RESPONSE",
      message: error?.message ?? "Unexpected response from the server.",
      details: error?.details,
      request_id: error?.request_id,
    });
  }
  return envelope.data;
}

/** Stores a session returned by sign-in, sign-up verify, Google or refresh. */
export function acceptSession(session: EstablishedSession): void {
  accessToken = session.access_token;
  accessExpiresAt = Date.parse(session.access_token_expires_at) || 0;
  setState({ status: "signed-in", user: session.user });
}

function clearLocal() {
  accessToken = null;
  accessExpiresAt = 0;
  setState({ status: "signed-out" });
}

/**
 * Gets a new access token with the refresh cookie. Resolves false when there is no valid
 * session (the learner must sign in); throws only for network or server failures, which keep
 * the current state so the page can offer a retry.
 */
export function refreshSession(): Promise<boolean> {
  inflight ??= (async () => {
    try {
      acceptSession(await sessionRequest<EstablishedSession>("refresh"));
      return true;
    } catch (err) {
      if (err instanceof ApiError && (err.status === 401 || err.status === 403)) {
        clearLocal();
        return false;
      }
      throw err;
    } finally {
      inflight = null;
    }
  })();
  return inflight;
}

/** A valid access token, refreshing first when needed. Null when signed out. */
export async function getAccessToken(): Promise<string | null> {
  if (accessToken && Date.now() < accessExpiresAt - EXPIRY_MARGIN_MS) return accessToken;
  const ok = await refreshSession();
  return ok ? accessToken : null;
}

/**
 * Runs an authenticated API call. On 401 it refreshes once and retries; if the session is gone
 * it signs out locally and rethrows, and the session boundary sends the learner to sign-in.
 *
 *   const me = await withAuth((token) => api().request("/auth/me", { accessToken: token }));
 */
export async function withAuth<T>(call: (accessToken: string) => Promise<T>): Promise<T> {
  const token = await getAccessToken();
  if (!token) throw new ApiError(401, { code: "UNAUTHORIZED", message: "Sign in to continue." });
  try {
    return await call(token);
  } catch (err) {
    if (!(err instanceof ApiError) || err.status !== 401) throw err;
    accessToken = null; // force a refresh even if the clock says the token is fresh
    if (!(await refreshSession()) || !accessToken) throw err;
    return call(accessToken);
  }
}

/** Revokes the session on the server (best effort) and clears it here. */
export async function signOut(): Promise<void> {
  try {
    await sessionRequest<void>("logout");
  } finally {
    clearLocal();
  }
}

/** Test helper: back to the initial state. */
export function resetSessionForTests(): void {
  accessToken = null;
  accessExpiresAt = 0;
  inflight = null;
  state = { status: "loading" };
  listeners.clear();
}
