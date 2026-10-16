"use client";

// The signed-in learner's account preferences (2.11), shared by the preferences sync, the shell
// wallpaper and the Settings page, so they are fetched once per session and update everywhere
// after a change. Appearance values (theme, accent, density, motion, sidebar) live in the device
// cookies and are pushed by PreferencesSync; this store is the account copy.

import { useCallback, useEffect, useSyncExternalStore } from "react";

import { useSession } from "@/features/auth/session";

import {
  fetchPreferences,
  updatePreferences,
  type Preferences,
  type PreferencesPatch,
  type WallpaperChoice,
  type WallpaperMode,
} from "./api";

export type PreferencesState =
  | { status: "idle" }
  | { status: "loading"; userId: string }
  | { status: "error"; userId: string; error: unknown }
  | { status: "ready"; userId: string; preferences: Preferences };

const IDLE: PreferencesState = { status: "idle" };
let state: PreferencesState = IDLE;
let inflight: { userId: string; promise: Promise<void> } | null = null;
const listeners = new Set<() => void>();

function setState(next: PreferencesState) {
  state = next;
  for (const l of listeners) l();
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function getPreferencesState(): PreferencesState {
  return state;
}

/** Fetches the preferences of userId (single-flight). */
export function loadPreferences(userId: string): Promise<void> {
  if (inflight?.userId === userId) return inflight.promise;
  setState({ status: "loading", userId });
  const promise = fetchPreferences()
    .then((preferences) => {
      if (inflight?.promise === promise) setState({ status: "ready", userId, preferences });
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

/** Stores preferences the API returned after a change. */
export function setPreferencesData(userId: string, preferences: Preferences): void {
  inflight = null;
  setState({ status: "ready", userId, preferences });
}

function readyFor(userId: string): Preferences | null {
  return state.status === "ready" && state.userId === userId ? state.preferences : null;
}

export interface WallpaperChange {
  mode?: WallpaperMode;
  /** New source for `mode`. */
  choice?: Pick<WallpaperChoice, "source" | "gallery_id">;
  opacity?: number;
  blur?: number;
}

function applyLocally(p: Preferences, change: WallpaperChange): Preferences {
  const wallpaper = { ...p.wallpaper };
  if (change.mode && change.choice) {
    wallpaper[change.mode] = {
      ...wallpaper[change.mode],
      source: change.choice.source,
      gallery_id: change.choice.source === "gallery" ? change.choice.gallery_id : null,
    };
  }
  if (change.opacity !== undefined) wallpaper.opacity = change.opacity;
  if (change.blur !== undefined) wallpaper.blur = change.blur;
  return { ...p, wallpaper };
}

/** Shows a wallpaper change right away (e.g. while a slider moves), without saving it. */
export function previewWallpaper(userId: string, change: WallpaperChange): void {
  const current = readyFor(userId);
  if (current) setState({ status: "ready", userId, preferences: applyLocally(current, change) });
}

/**
 * Applies a wallpaper change at once and saves it. If saving fails, `revertTo` (or the state
 * before the change) is restored and the error is rethrown for the caller to show.
 */
export async function saveWallpaper(
  userId: string,
  change: WallpaperChange,
  revertTo?: Preferences,
): Promise<void> {
  const before = revertTo ?? readyFor(userId);
  if (!before) return;
  previewWallpaper(userId, change);

  const patch: NonNullable<PreferencesPatch["wallpaper"]> = {};
  if (change.mode && change.choice) {
    patch[change.mode] =
      change.choice.source === "gallery" && change.choice.gallery_id
        ? { source: "gallery", gallery_id: change.choice.gallery_id }
        : { source: change.choice.source };
  }
  if (change.opacity !== undefined) patch.opacity = change.opacity;
  if (change.blur !== undefined) patch.blur = change.blur;

  try {
    const next = await updatePreferences({ wallpaper: patch });
    if (readyFor(userId)) setPreferencesData(userId, next);
  } catch (err) {
    if (readyFor(userId)) setPreferencesData(userId, before);
    throw err;
  }
}

/** Test helper. */
export function resetPreferencesForTests(): void {
  inflight = null;
  state = IDLE;
}

/**
 * The learner's account preferences. Loads them once the session is known and again when
 * another learner signs in; "idle" while there is no session.
 */
export function usePreferences(): { state: PreferencesState; reload: () => void } {
  const session = useSession();
  const current = useSyncExternalStore(subscribe, getPreferencesState, () => IDLE);
  const userId = session.status === "signed-in" ? session.user.id : null;

  useEffect(() => {
    if (!userId) {
      if (session.status === "signed-out" && getPreferencesState().status !== "idle") {
        inflight = null;
        setState(IDLE);
      }
      return;
    }
    const s = getPreferencesState();
    if (s.status === "idle" || s.userId !== userId) void loadPreferences(userId);
  }, [userId, session.status]);

  const reload = useCallback(() => {
    if (userId) {
      inflight = null;
      void loadPreferences(userId);
    }
  }, [userId]);

  const visible = userId && current.status !== "idle" && current.userId === userId ? current : IDLE;
  return { state: visible, reload };
}
