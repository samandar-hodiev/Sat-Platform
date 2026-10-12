// Learner profile API (session 2.01). Contract: docs/API.md → Profile.

import { api } from "@/lib/api/client";
import { apiUrl } from "@/lib/env";

import { withAuth } from "@/features/auth/session";

export interface LearnerProfile {
  email: string;
  first_name: string | null;
  last_name: string | null;
  /** E.164, e.g. +998901234567. */
  phone: string | null;
  /** YYYY-MM-DD. */
  date_of_birth: string | null;
  /** Path on the API origin; use avatarSrc() for an <img>. Null: the default avatar. */
  avatar_url: string | null;
  timezone: string;
  has_password: boolean;
  /** Profile Setup must set a password (email accounts without one). */
  password_required: boolean;
  profile_completed: boolean;
  profile_completed_at: string | null;
}

export interface ProfileSetupInput {
  first_name: string;
  last_name: string;
  phone: string;
  date_of_birth: string;
  timezone: string;
  /** Empty when the learner keeps signing in without a password (Google accounts). */
  password: string;
}

export function fetchProfile(): Promise<LearnerProfile> {
  return withAuth((token) => api().request<LearnerProfile>("/me/profile", { accessToken: token }));
}

export function completeProfileSetup(input: ProfileSetupInput): Promise<LearnerProfile> {
  return withAuth((token) =>
    api().request<LearnerProfile>("/me/profile/setup", {
      method: "PUT",
      body: input,
      accessToken: token,
    }),
  );
}

export function uploadAvatar(file: Blob): Promise<LearnerProfile> {
  return withAuth((token) => {
    // A fresh FormData per attempt: withAuth may retry after a token refresh.
    const form = new FormData();
    form.append("file", file);
    return api().request<LearnerProfile>("/me/avatar", {
      method: "POST",
      body: form,
      accessToken: token,
    });
  });
}

export function removeAvatar(): Promise<LearnerProfile> {
  return withAuth((token) =>
    api().request<LearnerProfile>("/me/avatar", { method: "DELETE", accessToken: token }),
  );
}

/** Full URL of an avatar path from the API, or null for the default avatar. */
export function avatarSrc(path: string | null | undefined): string | null {
  return path ? `${apiUrl()}${path}` : null;
}

/** "Ali Valiyev"; null while the learner has not entered a name. */
export function displayName(p: Pick<LearnerProfile, "first_name" | "last_name">): string | null {
  const name = [p.first_name, p.last_name].filter(Boolean).join(" ").trim();
  return name || null;
}
