// Theme preferences: pure helpers shared by the inline pre-paint script, ThemeProvider and tests.
//
// Preferences are stored in a first-party cookie (not localStorage: see eslint.config.mjs) so the
// choice survives reloads before login. Logged-in sync to the database is session 2.11's work
// (PRD §4.6); it can call ThemeProvider's setters with the stored values.
// Learner and Owner keep independent preferences (PRD §21.3), selected by the URL scope.

export const THEME_MODES = ["light", "dark", "system"] as const;
export type ThemeMode = (typeof THEME_MODES)[number];
export type ResolvedTheme = "light" | "dark";

export const ACCENTS = ["neutral", "blue", "indigo", "violet", "teal", "rose"] as const;
export type Accent = (typeof ACCENTS)[number];

export const DENSITIES = ["comfortable", "compact"] as const;
export type Density = (typeof DENSITIES)[number];

export const MOTIONS = ["system", "reduced", "full"] as const;
export type Motion = (typeof MOTIONS)[number];

export type ThemeScope = "learner" | "owner";

export interface ThemePreferences {
  mode: ThemeMode;
  accent: Accent;
  density: Density;
  motion: Motion;
}

/** PRD §4.2: first launch opens a neutral Light mode. */
export const DEFAULT_PREFERENCES: ThemePreferences = {
  mode: "light",
  accent: "neutral",
  density: "comfortable",
  motion: "system",
};

export const COOKIE_MAX_AGE_SECONDS = 60 * 60 * 24 * 365;

export function scopeForPath(pathname: string): ThemeScope {
  return pathname === "/owner" || pathname.startsWith("/owner/") ? "owner" : "learner";
}

export function cookieName(scope: ThemeScope): string {
  return `sat_${scope}_appearance`;
}

function pick<T extends string>(allowed: readonly T[], value: unknown, fallback: T): T {
  return typeof value === "string" && (allowed as readonly string[]).includes(value)
    ? (value as T)
    : fallback;
}

/** Cookie value format: "mode.accent.density.motion", e.g. "dark.blue.comfortable.system". */
export function serializePreferences(prefs: ThemePreferences): string {
  return [prefs.mode, prefs.accent, prefs.density, prefs.motion].join(".");
}

/** Never throws: unknown or tampered values fall back to defaults field by field. */
export function parsePreferences(raw: string | null | undefined): ThemePreferences {
  const [mode, accent, density, motion] = (raw ?? "").split(".");
  return {
    mode: pick(THEME_MODES, mode, DEFAULT_PREFERENCES.mode),
    accent: pick(ACCENTS, accent, DEFAULT_PREFERENCES.accent),
    density: pick(DENSITIES, density, DEFAULT_PREFERENCES.density),
    motion: pick(MOTIONS, motion, DEFAULT_PREFERENCES.motion),
  };
}

export function readCookie(cookieHeader: string, name: string): string | null {
  for (const part of cookieHeader.split(";")) {
    const [key, ...rest] = part.trim().split("=");
    if (key === name) {
      try {
        return decodeURIComponent(rest.join("="));
      } catch {
        return null;
      }
    }
  }
  return null;
}

export function resolveTheme(mode: ThemeMode, systemPrefersDark: boolean): ResolvedTheme {
  if (mode === "system") return systemPrefersDark ? "dark" : "light";
  return mode;
}

/** The attributes the CSS tokens key off (see globals.css). */
export function themeAttributes(
  prefs: ThemePreferences,
  systemPrefersDark: boolean,
): Record<"data-theme" | "data-accent" | "data-density" | "data-motion", string> {
  return {
    "data-theme": resolveTheme(prefs.mode, systemPrefersDark),
    "data-accent": prefs.accent,
    "data-density": prefs.density,
    "data-motion": prefs.motion,
  };
}

export function applyThemeAttributes(
  element: HTMLElement,
  prefs: ThemePreferences,
  systemPrefersDark: boolean,
): void {
  for (const [name, value] of Object.entries(themeAttributes(prefs, systemPrefersDark))) {
    element.setAttribute(name, value);
  }
}
