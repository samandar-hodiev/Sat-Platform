// @vitest-environment jsdom
import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { WallpaperChoice } from "../api";
import { GALLERY, GALLERY_IDS, isGalleryId } from "./gallery";
import { WallpaperView, wallpaperShownOn } from "./LearnerWallpaper";

vi.mock("@/lib/env", () => ({ apiUrl: () => "http://api.test" }));

const none: WallpaperChoice = { source: "none", gallery_id: null, custom_url: null };

describe("wallpaper gallery", () => {
  it("lists the same ids as gallery.json (the API allow-list)", () => {
    expect(GALLERY.map((g) => g.id)).toEqual(GALLERY_IDS);
  });

  it("recognises gallery ids", () => {
    expect(isGalleryId("aurora")).toBe(true);
    expect(isGalleryId("sunset")).toBe(false);
    expect(isGalleryId(1)).toBe(false);
  });
});

describe("wallpaperShownOn", () => {
  it.each([
    ["/app", true],
    ["/app/", true],
    ["/app/plan", true],
    ["/app/plan/week", true],
    ["/app/progress", true],
    ["/app/learn", false],
    ["/app/practice", false],
    ["/app/library", false],
    ["/app/assistant", false],
    ["/app/settings", false],
    ["/app/planner", false],
    ["/exam/123", false],
  ])("%s → %s", (path, shown) => {
    expect(wallpaperShownOn(path)).toBe(shown);
  });
});

describe("WallpaperView", () => {
  it("renders nothing without a wallpaper", () => {
    const { container } = render(<WallpaperView choice={none} opacity={40} blur={0} />);
    expect(container.innerHTML).toBe("");
  });

  it("renders nothing for a custom choice without its image", () => {
    const { container } = render(
      <WallpaperView
        choice={{ source: "custom", gallery_id: null, custom_url: null }}
        opacity={40}
        blur={0}
      />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("draws a gallery pattern with the chosen opacity, hidden from assistive tech", () => {
    render(
      <WallpaperView
        choice={{ source: "gallery", gallery_id: "dots", custom_url: null }}
        opacity={60}
        blur={0}
      />,
    );
    const layer = screen.getByTestId("wallpaper");
    expect(layer.getAttribute("aria-hidden")).toBe("true");
    expect(layer.classList.contains("fixed")).toBe(true);
    expect(layer.classList.contains("pointer-events-none")).toBe(true);
    screen.getByTestId("gallery-pattern-dots");
    const inner = layer.firstElementChild as HTMLElement;
    expect(inner.style.opacity).toBe("0.6");
    expect(inner.style.filter).toBe("");
  });

  it("shows an uploaded image from the API origin, blurred", () => {
    render(
      <WallpaperView
        choice={{
          source: "custom",
          gallery_id: null,
          custom_url: "/api/v1/media/wallpapers/u/f.jpg",
        }}
        opacity={40}
        blur={8}
        position="absolute"
      />,
    );
    const layer = screen.getByTestId("wallpaper");
    expect(layer.classList.contains("absolute")).toBe(true);
    const inner = layer.firstElementChild as HTMLElement;
    expect(inner.style.filter).toBe("blur(8px)");
    expect((inner.firstElementChild as HTMLElement).style.backgroundImage).toContain(
      "http://api.test/api/v1/media/wallpapers/u/f.jpg",
    );
  });

  it.each(GALLERY.map((g) => g.id))("draws the %s pattern", (id) => {
    render(
      <WallpaperView
        choice={{ source: "gallery", gallery_id: id, custom_url: null }}
        opacity={40}
        blur={0}
      />,
    );
    expect(screen.getByTestId(`gallery-pattern-${id}`).querySelector("*")).not.toBeNull();
  });
});
