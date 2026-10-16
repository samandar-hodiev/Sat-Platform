// Collapsed/expanded desktop sidebar, kept in a first-party cookie so the server renders the
// right width on the first paint (no layout jump). Not localStorage: see eslint.config.mjs.
// Session 2.11 syncs it with the account's preferences (PRD §20): writes notify subscribers, so
// a value loaded from the account updates the shell too.

import { COOKIE_MAX_AGE_SECONDS, readCookie } from "@/lib/theme/theme";

export const SIDEBAR_COOKIE = "sat_learner_sidebar";

export type SidebarState = "expanded" | "collapsed";

/** Never throws: anything but "collapsed" means expanded (the default). */
export function parseSidebarState(raw: string | null | undefined): SidebarState {
  return raw === "collapsed" ? "collapsed" : "expanded";
}

export function sidebarCookie(state: SidebarState): string {
  return `${SIDEBAR_COOKIE}=${state}; Path=/; Max-Age=${COOKIE_MAX_AGE_SECONDS}; SameSite=Lax`;
}

const listeners = new Set<() => void>();

export function subscribeSidebarState(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

/** The state stored on this device; null when no choice was stored yet. */
export function readSidebarState(): SidebarState | null {
  const raw = readCookie(document.cookie, SIDEBAR_COOKIE);
  return raw === null ? null : parseSidebarState(raw);
}

export function writeSidebarState(state: SidebarState): void {
  document.cookie = sidebarCookie(state);
  listeners.forEach((listener) => listener());
}
