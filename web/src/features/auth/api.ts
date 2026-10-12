// Learner auth API (session 1.01). Contract: docs/API.md → Auth, docs/AUTHENTICATION.md.
//
// Calls that create a session (sign-in, sign-up verify, Google) go through the Next.js session
// route handlers (/api/session/*), so the refresh token lands in an HttpOnly cookie and never
// reaches browser JavaScript; the access token is kept in memory by ./session. The other calls
// go straight to the Go API.

import { api } from "@/lib/api/client";

import {
  acceptSession,
  sessionRequest,
  withAuth,
  type EstablishedSession,
  type SessionUser,
} from "./session";

export type AuthUser = SessionUser;
export type SessionResult = EstablishedSession;

export interface CodeSent {
  /** When the code stops working (ISO 8601). */
  expires_at: string;
  /** Earliest time another code can be requested (ISO 8601). */
  resend_available_at: string;
}

async function establish(action: string, body: unknown): Promise<SessionResult> {
  const session = await sessionRequest<SessionResult>(action, body);
  acceptSession(session);
  return session;
}

export function signIn(input: { email: string; password: string; remember: boolean }) {
  return establish("sign-in", input);
}

export function signInWithGoogle(input: { id_token: string }) {
  return establish("google", input);
}

/** Sends a verification code. The answer is the same whether or not the email is registered. */
export function startSignUp(input: { email: string }) {
  return api().request<CodeSent>("/auth/sign-up/start", { method: "POST", body: input });
}

/**
 * Verifies the code and signs the learner in. A new address gets an account
 * (is_new_user: true → Profile Setup); an existing learner is simply signed in.
 */
export function verifySignUp(input: { email: string; code: string }) {
  return establish("sign-up/verify", input);
}

/** Always succeeds for a well-formed email, so it cannot reveal which emails have accounts. */
export function requestPasswordReset(input: { email: string }) {
  return api().request<void>("/auth/password/forgot", { method: "POST", body: input });
}

export function resetPassword(input: { token: string; password: string }) {
  return api().request<void>("/auth/password/reset", { method: "POST", body: input });
}

/** The signed-in user (GET /auth/me). */
export function fetchMe() {
  return withAuth((token) =>
    api().request<{ user: AuthUser }>("/auth/me", { accessToken: token }),
  ).then((r) => r.user);
}
