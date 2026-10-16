// Built-in wallpaper gallery (PRD §4.5). The patterns are drawn in patterns.tsx with theme tokens,
// so each one follows the colour mode and the accent. gallery.json is the id list the API accepts
// (api/internal/preferences checks that both sides list the same ids).

import ids from "./gallery.json";

export type GalleryId = "aurora" | "dots" | "grid" | "waves" | "contours" | "peaks";

export const GALLERY: readonly { id: GalleryId; label: string }[] = [
  { id: "aurora", label: "Aurora" },
  { id: "dots", label: "Dots" },
  { id: "grid", label: "Grid" },
  { id: "waves", label: "Waves" },
  { id: "contours", label: "Contours" },
  { id: "peaks", label: "Peaks" },
];

export const GALLERY_IDS: readonly string[] = ids;

export function isGalleryId(value: unknown): value is GalleryId {
  return typeof value === "string" && GALLERY.some((g) => g.id === value);
}
