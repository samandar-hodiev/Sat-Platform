// Learner preferences API (session 2.11). Contract: docs/API.md → Preferences.

import { api } from "@/lib/api/client";
import { apiUrl } from "@/lib/env";
import type { Accent, Density, Motion, ThemeMode } from "@/lib/theme/theme";

import { withAuth } from "@/features/auth/session";

import type { GalleryId } from "./wallpaper/gallery";

export type WallpaperMode = "light" | "dark";
export type WallpaperSource = "none" | "gallery" | "custom";

export interface WallpaperChoice {
  source: WallpaperSource;
  /** Set only when source is "gallery". */
  gallery_id: GalleryId | null;
  /** The uploaded image of this mode (path on the API origin), also while another source is chosen. */
  custom_url: string | null;
}

export interface Preferences {
  theme: ThemeMode;
  accent: Accent;
  density: Density;
  motion: Motion;
  sidebar_collapsed: boolean;
  wallpaper: {
    light: WallpaperChoice;
    dark: WallpaperChoice;
    /** Percent the wallpaper shows through the page background, 10–80. */
    opacity: number;
    /** Blur radius in CSS pixels, 0–20. */
    blur: number;
  };
  /** False while nothing was ever saved: the values are the defaults. */
  stored: boolean;
  updated_at: string | null;
}

export interface WallpaperChoiceInput {
  source: WallpaperSource;
  gallery_id?: GalleryId;
}

/** Only the fields sent change. */
export interface PreferencesPatch {
  theme?: ThemeMode;
  accent?: Accent;
  density?: Density;
  motion?: Motion;
  sidebar_collapsed?: boolean;
  wallpaper?: {
    light?: WallpaperChoiceInput;
    dark?: WallpaperChoiceInput;
    opacity?: number;
    blur?: number;
  };
}

export const OPACITY_MIN = 10;
export const OPACITY_MAX = 80;
export const BLUR_MAX = 20;
/** Upload limit (the API checks it too). */
export const WALLPAPER_MAX_BYTES = 8 * 1024 * 1024;
export const WALLPAPER_TYPES = ["image/jpeg", "image/png", "image/webp"];

export function fetchPreferences(): Promise<Preferences> {
  return withAuth((token) => api().request<Preferences>("/me/preferences", { accessToken: token }));
}

export function updatePreferences(patch: PreferencesPatch): Promise<Preferences> {
  return withAuth((token) =>
    api().request<Preferences>("/me/preferences", {
      method: "PATCH",
      body: patch,
      accessToken: token,
    }),
  );
}

export function uploadWallpaper(mode: WallpaperMode, file: Blob): Promise<Preferences> {
  return withAuth((token) => {
    // A fresh FormData per attempt: withAuth may retry after a token refresh.
    const form = new FormData();
    form.append("file", file);
    return api().request<Preferences>(`/me/wallpapers/${mode}`, {
      method: "POST",
      body: form,
      accessToken: token,
    });
  });
}

export function removeWallpaper(mode: WallpaperMode): Promise<Preferences> {
  return withAuth((token) =>
    api().request<Preferences>(`/me/wallpapers/${mode}`, { method: "DELETE", accessToken: token }),
  );
}

/** Full URL of a wallpaper path from the API. */
export function wallpaperSrc(path: string): string {
  return `${apiUrl()}${path}`;
}
