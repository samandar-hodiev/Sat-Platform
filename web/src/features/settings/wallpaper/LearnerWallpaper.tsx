"use client";

import { usePathname } from "next/navigation";
import type { CSSProperties } from "react";

import { useTheme } from "@/components/theme/ThemeProvider";
import { cn } from "@/lib/cn";

import { wallpaperSrc, type WallpaperChoice } from "../api";
import { usePreferences } from "../preferencesStore";
import { GalleryPattern } from "./patterns";

/**
 * Pages that show the wallpaper (decided for 2.11: Dashboard, My Study Plan, My Progress).
 * Reading and practice pages, the AI chat and the exam stay plain so text is never harder to
 * read (PRD §4.5); the exam layout has no shell at all.
 */
export function wallpaperShownOn(pathname: string): boolean {
  const path = pathname.replace(/\/+$/, "");
  return (
    path === "/app" ||
    ["/app/plan", "/app/progress"].some((base) => path === base || path.startsWith(`${base}/`))
  );
}

export interface WallpaperViewProps {
  choice: WallpaperChoice;
  /** 10–80: how much the wallpaper shows through the page background. */
  opacity: number;
  blur: number;
  /** "fixed" behind the whole page (shell), "absolute" inside a preview box. */
  position?: "fixed" | "absolute";
  className?: string;
}

/** The wallpaper layer itself; renders nothing for "none" or a choice without its image. */
export function WallpaperView({
  choice,
  opacity,
  blur,
  position = "fixed",
  className,
}: WallpaperViewProps) {
  const gallery = choice.source === "gallery" ? choice.gallery_id : null;
  const custom = choice.source === "custom" ? choice.custom_url : null;
  if (!gallery && !custom) return null;

  // User-chosen values, so inline styles; the canvas colour behind keeps text readable.
  const style: CSSProperties = { opacity: opacity / 100 };
  if (blur > 0) {
    style.filter = `blur(${blur}px)`;
    // Blur fades the edges toward transparent; scaling up a little hides that.
    style.transform = "scale(1.06)";
  }

  return (
    <div
      aria-hidden="true"
      data-testid="wallpaper"
      data-source={choice.source}
      className={cn(
        "pointer-events-none inset-0 z-(--z-base) overflow-hidden",
        position === "fixed" ? "fixed" : "absolute",
        className,
      )}
    >
      <div className="absolute inset-0" style={style}>
        {gallery ? (
          <GalleryPattern id={gallery} />
        ) : custom ? (
          <div
            className="size-full bg-cover bg-center"
            style={{ backgroundImage: `url("${wallpaperSrc(custom)}")` }}
          />
        ) : null}
      </div>
    </div>
  );
}

/** The signed-in learner's wallpaper for the current colour mode, on the pages that show one. */
export function LearnerWallpaper() {
  const pathname = usePathname() ?? "";
  const { resolvedTheme } = useTheme();
  const { state } = usePreferences();
  if (state.status !== "ready" || !wallpaperShownOn(pathname)) return null;
  const { wallpaper } = state.preferences;
  return (
    <WallpaperView
      choice={wallpaper[resolvedTheme]}
      opacity={wallpaper.opacity}
      blur={wallpaper.blur}
    />
  );
}
