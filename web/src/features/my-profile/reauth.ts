// Confirming the learner before a sensitive change (2.10): the current password, or a code sent
// to the current email when the account has no password. Rules match api/internal/auth/account.go.

import { CODE_LENGTH } from "@/features/auth/validation";

import type { Reauth, SignInMethods } from "./api";

export type ReauthMethod = SignInMethods["reauth_method"];

export interface ReauthValues {
  password: string;
  code: string;
}

export const EMPTY_REAUTH: ReauthValues = { password: "", code: "" };

/** Client-side check; keys are the API field names. */
export function validateReauth(
  method: ReauthMethod,
  values: ReauthValues,
): Partial<Record<"current_password" | "code", string>> {
  if (method === "password") {
    return values.password ? {} : { current_password: "Enter your current password." };
  }
  if (!values.code) return { code: "Enter the code we sent to your email." };
  if (values.code.length !== CODE_LENGTH) return { code: `Enter the ${CODE_LENGTH}-digit code.` };
  return {};
}

/** The request fields for the method: only the one that applies is sent. */
export function reauthBody(method: ReauthMethod, values: ReauthValues): Reauth {
  return method === "password" ? { current_password: values.password } : { code: values.code };
}

/** Whole seconds until an ISO time, never negative. */
export function secondsUntil(iso: string, now: number = Date.now()): number {
  return Math.max(0, Math.ceil((Date.parse(iso) - now) / 1000));
}
