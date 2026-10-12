"use client";

// The signed-in learner's profile, shared by the Profile Setup page, the /app gate and the
// shell header, so it is fetched once per session and updates everywhere after a change.

import { useCallback, useEffect, useSyncExternalStore } from "react";

import { useSession } from "@/features/auth/session";

import { fetchProfile, type LearnerProfile } from "./api";

export type ProfileState =
  | { status: "idle" }
  | { status: "loading"; userId: string }
  | { status: "error"; userId: string; error: unknown }
  | { status: "ready"; userId: string; profile: LearnerProfile };

let state: ProfileState = { status: "idle" };
let inflight: { userId: string; promise: Promise<void> } | null = null;
const listeners = new Set<() => void>();
const IDLE: ProfileState = { status: "idle" };

function setState(next: ProfileState) {
  state = next;
  for (const l of listeners) l();
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function getProfileState(): ProfileState {
  return state;
}

/** Fetches the profile of userId (single-flight). */
export function loadProfile(userId: string): Promise<void> {
  if (inflight?.userId === userId) return inflight.promise;
  setState({ status: "loading", userId });
  const promise = fetchProfile()
    .then((profile) => {
      if (inflight?.promise === promise) setState({ status: "ready", userId, profile });
    })
    .catch((error: unknown) => {
      if (inflight?.promise === promise) setState({ status: "error", userId, error });
    })
    .finally(() => {
      if (inflight?.promise === promise) inflight = null;
    });
  inflight = { userId, promise };
  return promise;
}

/** Stores a profile the API returned after a change (setup, avatar). */
export function setProfile(userId: string, profile: LearnerProfile): void {
  inflight = null;
  setState({ status: "ready", userId, profile });
}

/** Test helper. */
export function resetProfileForTests(): void {
  inflight = null;
  state = IDLE;
}

/**
 * The learner's profile. Loads it once the session is known and again when another learner
 * signs in; "idle" while there is no session.
 */
export function useLearnerProfile(): { state: ProfileState; reload: () => void } {
  const session = useSession();
  const current = useSyncExternalStore(subscribe, getProfileState, () => IDLE);
  const userId = session.status === "signed-in" ? session.user.id : null;

  useEffect(() => {
    if (!userId) {
      if (session.status === "signed-out" && getProfileState().status !== "idle") {
        inflight = null;
        setState(IDLE);
      }
      return;
    }
    const s = getProfileState();
    if (s.status === "idle" || s.userId !== userId) void loadProfile(userId);
  }, [userId, session.status]);

  const reload = useCallback(() => {
    if (userId) {
      inflight = null;
      void loadProfile(userId);
    }
  }, [userId]);

  const visible = userId && current.status !== "idle" && current.userId === userId ? current : IDLE;
  return { state: visible, reload };
}
