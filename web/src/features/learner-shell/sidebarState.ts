// Collapsed/expanded desktop sidebar, kept in a first-party cookie so the server renders the
// right width on the first paint (no layout jump). Not localStorage: see eslint.config.mjs.
// Session 2.11 may sync it with the account's layout preferences (PRD §20).

import { COOKIE_MAX_AGE_SECONDS } from "@/lib/theme/theme";

export const SIDEBAR_COOKIE = "sat_learner_sidebar";

export type SidebarState = "expanded" | "collapsed";

/** Never throws: anything but "collapsed" means expanded (the default). */
export function parseSidebarState(raw: string | null | undefined): SidebarState {
  return raw === "collapsed" ? "collapsed" : "expanded";
}

export function sidebarCookie(state: SidebarState): string {
  return `${SIDEBAR_COOKIE}=${state}; Path=/; Max-Age=${COOKIE_MAX_AGE_SECONDS}; SameSite=Lax`;
}

export function writeSidebarState(state: SidebarState): void {
  document.cookie = sidebarCookie(state);
}
