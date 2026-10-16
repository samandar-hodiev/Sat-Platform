// My Profile API (session 2.10). Contract: docs/API.md → My Profile.

import { api } from "@/lib/api/client";

import type { CodeSent } from "@/features/auth/api";
import { withAuth } from "@/features/auth/session";
import type { LearnerProfile } from "@/features/profile/api";

export type { CodeSent };

export interface SignInMethods {
  email: string;
  email_verified: boolean;
  has_password: boolean;
  /** Signing in with a code sent to the email always works. */
  email_code: true;
  /** What sensitive changes ask for. */
  reauth_method: "password" | "email_code";
  google: { email: string | null; linked_at: string } | null;
}

/**
 * Confirms a sensitive change: the current password when the account has one, otherwise a code
 * from sendReauthCode(). Only the field that applies is sent.
 */
export interface Reauth {
  current_password?: string;
  code?: string;
}

/** Absent fields stay as they are; phone "" removes the phone. */
export interface ProfileUpdate {
  first_name?: string;
  last_name?: string;
  phone?: string;
  date_of_birth?: string;
  timezone?: string;
}

export interface EmailChangeStarted extends CodeSent {
  new_email: string;
}

export function updateProfile(input: ProfileUpdate): Promise<LearnerProfile> {
  return withAuth((token) =>
    api().request<LearnerProfile>("/me/profile", {
      method: "PATCH",
      body: input,
      accessToken: token,
    }),
  );
}

export function fetchSignInMethods(): Promise<SignInMethods> {
  return withAuth((token) =>
    api().request<SignInMethods>("/me/sign-in-methods", { accessToken: token }),
  );
}

export function sendReauthCode(): Promise<CodeSent> {
  return withAuth((token) =>
    api().request<CodeSent>("/me/reauth/code", { method: "POST", accessToken: token }),
  );
}

export function startEmailChange(
  input: { new_email: string } & Reauth,
): Promise<EmailChangeStarted> {
  return withAuth((token) =>
    api().request<EmailChangeStarted>("/me/email/change", {
      method: "POST",
      body: input,
      accessToken: token,
    }),
  );
}

export function confirmEmailChange(input: { code: string }): Promise<SignInMethods> {
  return withAuth((token) =>
    api().request<SignInMethods>("/me/email/change/confirm", {
      method: "POST",
      body: input,
      accessToken: token,
    }),
  );
}

export function changePassword(input: { new_password: string } & Reauth): Promise<void> {
  return withAuth((token) =>
    api().request<void>("/me/password", { method: "PUT", body: input, accessToken: token }),
  );
}

export function linkGoogle(input: { id_token: string } & Reauth): Promise<SignInMethods> {
  return withAuth((token) =>
    api().request<SignInMethods>("/me/sign-in-methods/google", {
      method: "POST",
      body: input,
      accessToken: token,
    }),
  );
}

export function unlinkGoogle(): Promise<SignInMethods> {
  return withAuth((token) =>
    api().request<SignInMethods>("/me/sign-in-methods/google", {
      method: "DELETE",
      accessToken: token,
    }),
  );
}

export function deleteAccount(input: { confirmation: string } & Reauth): Promise<void> {
  return withAuth((token) =>
    api().request<void>("/me/account", { method: "DELETE", body: input, accessToken: token }),
  );
}
