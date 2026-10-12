"use client";

// The signed-in learner's onboarding answers, shared by the onboarding page and the /app gate
// (and the dashboard goal block, 2.04), so they are fetched once per session and update
// everywhere after a change.

import { useCallback, useEffect, useSyncExternalStore } from "react";

import { useSession } from "@/features/auth/session";

import { fetchOnboarding, type Onboarding } from "./api";

export type OnboardingState =
  | { status: "idle" }
  | { status: "loading"; userId: string }
  | { status: "error"; userId: string; error: unknown }
  | { status: "ready"; userId: string; onboarding: Onboarding };

const IDLE: OnboardingState = { status: "idle" };
let state: OnboardingState = IDLE;
let inflight: { userId: string; promise: Promise<void> } | null = null;
const listeners = new Set<() => void>();

function setState(next: OnboardingState) {
  state = next;
  for (const l of listeners) l();
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function getOnboardingState(): OnboardingState {
  return state;
}

/** Fetches the onboarding state of userId (single-flight). */
export function loadOnboarding(userId: string): Promise<void> {
  if (inflight?.userId === userId) return inflight.promise;
  setState({ status: "loading", userId });
  const promise = fetchOnboarding()
    .then((onboarding) => {
      if (inflight?.promise === promise) setState({ status: "ready", userId, onboarding });
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

/** Stores the state the API returned after a change (a saved step, completion). */
export function setOnboarding(userId: string, onboarding: Onboarding): void {
  inflight = null;
  setState({ status: "ready", userId, onboarding });
}

/** Test helper. */
export function resetOnboardingForTests(): void {
  inflight = null;
  state = IDLE;
}

/**
 * The learner's onboarding answers. Loads them once the session is known and again when
 * another learner signs in; "idle" while there is no session.
 */
export function useOnboarding(): { state: OnboardingState; reload: () => void } {
  const session = useSession();
  const current = useSyncExternalStore(subscribe, getOnboardingState, () => IDLE);
  const userId = session.status === "signed-in" ? session.user.id : null;

  useEffect(() => {
    if (!userId) {
      if (session.status === "signed-out" && getOnboardingState().status !== "idle") {
        inflight = null;
        setState(IDLE);
      }
      return;
    }
    const s = getOnboardingState();
    if (s.status === "idle" || s.userId !== userId) void loadOnboarding(userId);
  }, [userId, session.status]);

  const reload = useCallback(() => {
    if (userId) {
      inflight = null;
      void loadOnboarding(userId);
    }
  }, [userId]);

  const visible = userId && current.status !== "idle" && current.userId === userId ? current : IDLE;
  return { state: visible, reload };
}
