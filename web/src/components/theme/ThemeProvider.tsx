"use client";

import { usePathname } from "next/navigation";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useSyncExternalStore,
} from "react";

import {
  COOKIE_MAX_AGE_SECONDS,
  applyThemeAttributes,
  cookieName,
  parsePreferences,
  readCookie,
  resolveTheme,
  scopeForPath,
  serializePreferences,
  type Accent,
  type Density,
  type Motion,
  type ResolvedTheme,
  type ThemeMode,
  type ThemePreferences,
  type ThemeScope,
} from "@/lib/theme/theme";

interface ThemeContextValue extends ThemePreferences {
  scope: ThemeScope;
  /** The theme actually shown: "system" resolved against the OS setting. */
  resolvedTheme: ResolvedTheme;
  setMode: (mode: ThemeMode) => void;
  setAccent: (accent: Accent) => void;
  setDensity: (density: Density) => void;
  setMotion: (motion: Motion) => void;
  /** Replace all preferences at once, e.g. with values loaded from the account (2.11). */
  setPreferences: (prefs: ThemePreferences) => void;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

const DARK_QUERY = "(prefers-color-scheme: dark)";

// --- Cookie store: the cookie is the single source of truth for stored preferences.
const cookieListeners = new Set<() => void>();

function subscribeCookie(listener: () => void) {
  cookieListeners.add(listener);
  return () => cookieListeners.delete(listener);
}

function readStored(scope: ThemeScope): string | null {
  return readCookie(document.cookie, cookieName(scope));
}

function writeStored(scope: ThemeScope, prefs: ThemePreferences): void {
  const secure = window.location.protocol === "https:" ? "; Secure" : "";
  document.cookie =
    `${cookieName(scope)}=${encodeURIComponent(serializePreferences(prefs))}` +
    `; Path=/; Max-Age=${COOKIE_MAX_AGE_SECONDS}; SameSite=Lax${secure}`;
  cookieListeners.forEach((listener) => listener());
}

// --- OS color scheme store.
function subscribeSystem(listener: () => void) {
  const query = window.matchMedia(DARK_QUERY);
  query.addEventListener("change", listener);
  return () => query.removeEventListener("change", listener);
}

function systemPrefersDark(): boolean {
  return window.matchMedia(DARK_QUERY).matches;
}

/**
 * Owns the appearance attributes on <html>. The inline theme script applies them before paint;
 * this keeps them in sync with user changes, the OS theme and Learner ↔ Owner navigation.
 * During server render and hydration the context reports defaults, then switches to the stored
 * values; the DOM attributes are always computed from the cookie, so they never flash.
 */
export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const scope = scopeForPath(usePathname() ?? "/");

  const raw = useSyncExternalStore(
    subscribeCookie,
    () => readStored(scope),
    () => null,
  );
  const systemDark = useSyncExternalStore(subscribeSystem, systemPrefersDark, () => false);
  const prefs = useMemo(() => parsePreferences(raw), [raw]);

  useEffect(() => {
    applyThemeAttributes(
      document.documentElement,
      parsePreferences(readStored(scope)),
      systemPrefersDark(),
    );
  }, [scope, raw, systemDark]);

  const setPreferences = useCallback((next: ThemePreferences) => writeStored(scope, next), [scope]);

  const value = useMemo<ThemeContextValue>(
    () => ({
      ...prefs,
      scope,
      resolvedTheme: resolveTheme(prefs.mode, systemDark),
      setPreferences,
      setMode: (mode) => setPreferences({ ...prefs, mode }),
      setAccent: (accent) => setPreferences({ ...prefs, accent }),
      setDensity: (density) => setPreferences({ ...prefs, density }),
      setMotion: (motion) => setPreferences({ ...prefs, motion }),
    }),
    [prefs, scope, systemDark, setPreferences],
  );

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): ThemeContextValue {
  const context = useContext(ThemeContext);
  if (!context) throw new Error("useTheme must be used inside <ThemeProvider>.");
  return context;
}
