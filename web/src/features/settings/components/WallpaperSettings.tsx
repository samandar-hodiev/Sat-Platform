"use client";

import { useEffect, useId, useRef, useState } from "react";

import { useTheme } from "@/components/theme/ThemeProvider";
import { Button, ErrorState, Modal, Skeleton, useToast } from "@/components/ui";
import { ApiError } from "@/lib/api/client";
import { cn } from "@/lib/cn";

import {
  BLUR_MAX,
  OPACITY_MAX,
  OPACITY_MIN,
  WALLPAPER_MAX_BYTES,
  WALLPAPER_TYPES,
  removeWallpaper,
  uploadWallpaper,
  wallpaperSrc,
  type Preferences,
  type WallpaperChoice,
  type WallpaperMode,
} from "../api";
import {
  previewWallpaper,
  saveWallpaper,
  setPreferencesData,
  usePreferences,
  type WallpaperChange,
} from "../preferencesStore";
import { GALLERY, type GalleryId } from "../wallpaper/gallery";
import { WallpaperView } from "../wallpaper/LearnerWallpaper";
import { GalleryPattern } from "../wallpaper/patterns";
import { Section } from "./Section";

/** Debounce for the sliders: the preview follows at once, the save waits for a pause. */
export const SLIDER_SAVE_DELAY_MS = 400;

type Option = { kind: "none" } | { kind: "gallery"; id: GalleryId } | { kind: "custom" };

function selected(choice: WallpaperChoice): string {
  if (choice.source === "gallery" && choice.gallery_id) return `gallery:${choice.gallery_id}`;
  return choice.source;
}

function errorMessage(err: unknown, fallback: string): string {
  if (err instanceof ApiError) {
    const field = err.fieldErrors.file;
    if (field) return field;
    if (
      err.code === "PAYLOAD_TOO_LARGE" ||
      err.code === "RATE_LIMITED" ||
      err.code === "NETWORK_ERROR"
    ) {
      return err.message;
    }
  }
  return fallback;
}

/** Settings → Appearance → Wallpaper (PRD §4.5): gallery or own image per mode, opacity, blur. */
export function WallpaperSettings() {
  const { state, reload } = usePreferences();
  const { resolvedTheme, accent } = useTheme();
  const [mode, setMode] = useState<WallpaperMode>(resolvedTheme);

  return (
    <Section
      title="Wallpaper"
      description="Shown behind your Dashboard, Study Plan and Progress. Lessons, practice and exams always keep a plain background."
    >
      {state.status === "ready" ? (
        <WallpaperEditor
          userId={state.userId}
          prefs={state.preferences}
          mode={mode}
          onModeChange={setMode}
          accent={accent}
        />
      ) : state.status === "error" ? (
        <ErrorState
          title="Wallpaper settings could not be loaded"
          description="Check your connection and try again."
          onRetry={reload}
        />
      ) : (
        <div
          className="flex flex-col gap-3"
          aria-busy="true"
          aria-label="Loading wallpaper settings"
        >
          <Skeleton className="h-40 w-full rounded-lg" />
          <Skeleton className="h-24 w-full rounded-lg" />
        </div>
      )}
    </Section>
  );
}

function WallpaperEditor({
  userId,
  prefs,
  mode,
  onModeChange,
  accent,
}: {
  userId: string;
  prefs: Preferences;
  mode: WallpaperMode;
  onModeChange: (m: WallpaperMode) => void;
  accent: string;
}) {
  const { toast } = useToast();
  const choice = prefs.wallpaper[mode];
  const [busy, setBusy] = useState<"upload" | "remove" | null>(null);
  const [uploadError, setUploadError] = useState<string>();
  const [confirmRemove, setConfirmRemove] = useState(false);
  const fileInput = useRef<HTMLInputElement>(null);
  const name = useId();

  async function choose(option: Option) {
    if (option.kind === "custom" && !choice.custom_url) {
      fileInput.current?.click();
      return;
    }
    const change: WallpaperChange = {
      mode,
      choice:
        option.kind === "gallery"
          ? { source: "gallery", gallery_id: option.id }
          : { source: option.kind, gallery_id: null },
    };
    try {
      await saveWallpaper(userId, change);
    } catch {
      toast({ variant: "error", title: "Wallpaper not saved", description: "Please try again." });
    }
  }

  async function upload(file: File) {
    setUploadError(undefined);
    if (!WALLPAPER_TYPES.includes(file.type)) {
      setUploadError("Upload a JPG, PNG or WebP image.");
      return;
    }
    if (file.size > WALLPAPER_MAX_BYTES) {
      setUploadError("The image is larger than 8 MB.");
      return;
    }
    setBusy("upload");
    try {
      setPreferencesData(userId, await uploadWallpaper(mode, file));
    } catch (err) {
      setUploadError(errorMessage(err, "The image could not be uploaded. Please try again."));
    } finally {
      setBusy(null);
    }
  }

  async function remove() {
    setBusy("remove");
    try {
      setPreferencesData(userId, await removeWallpaper(mode));
      setConfirmRemove(false);
    } catch (err) {
      toast({
        variant: "error",
        title: "Image not removed",
        description: errorMessage(err, "Please try again."),
      });
    } finally {
      setBusy(null);
    }
  }

  const current = selected(choice);
  const modeLabel = mode === "light" ? "Light" : "Dark";

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div
          role="radiogroup"
          aria-label="Wallpaper for"
          className="inline-flex gap-0.5 rounded-lg border border-border bg-surface-muted p-0.5"
        >
          {(["light", "dark"] as const).map((m) => (
            <button
              key={m}
              type="button"
              role="radio"
              aria-checked={mode === m}
              onClick={() => onModeChange(m)}
              data-testid={`wallpaper-mode-${m}`}
              className={cn(
                "focus-ring inline-flex h-(--control-sm) items-center rounded-md px-3 text-sm font-medium",
                "transition-colors duration-(--duration-fast)",
                mode === m ? "bg-surface text-fg shadow-xs" : "text-fg-muted hover:text-fg",
              )}
            >
              {m === "light" ? "Light mode" : "Dark mode"}
            </button>
          ))}
        </div>
        <p className="text-sm text-fg-muted">Each mode has its own wallpaper.</p>
      </div>

      <Preview
        mode={mode}
        accent={accent}
        choice={choice}
        opacity={prefs.wallpaper.opacity}
        blur={prefs.wallpaper.blur}
      />

      <fieldset className="flex min-w-0 flex-col gap-2">
        <legend className="text-sm font-medium text-fg">{modeLabel} mode wallpaper</legend>
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <Tile
            name={name}
            value="none"
            checked={current === "none"}
            label="None"
            onSelect={() => void choose({ kind: "none" })}
          >
            <span className="flex size-full items-center justify-center bg-canvas text-xs text-fg-subtle">
              No wallpaper
            </span>
          </Tile>
          {GALLERY.map((g) => (
            <Tile
              key={g.id}
              name={name}
              value={`gallery:${g.id}`}
              checked={current === `gallery:${g.id}`}
              label={g.label}
              onSelect={() => void choose({ kind: "gallery", id: g.id })}
            >
              <span className="block size-full bg-canvas">
                <GalleryPattern id={g.id} />
              </span>
            </Tile>
          ))}
          {choice.custom_url ? (
            <Tile
              name={name}
              value="custom"
              checked={current === "custom"}
              label="Your image"
              onSelect={() => void choose({ kind: "custom" })}
            >
              <span
                className="block size-full bg-cover bg-center"
                style={{ backgroundImage: `url("${wallpaperSrc(choice.custom_url)}")` }}
              />
            </Tile>
          ) : null}
        </div>
      </fieldset>

      <div className="flex flex-col gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <input
            ref={fileInput}
            type="file"
            accept={WALLPAPER_TYPES.join(",")}
            className="sr-only"
            tabIndex={-1}
            aria-hidden="true"
            data-testid="wallpaper-file"
            onChange={(e) => {
              const file = e.target.files?.[0];
              e.target.value = "";
              if (file) void upload(file);
            }}
          />
          <Button
            variant="outline"
            size="sm"
            loading={busy === "upload"}
            disabled={busy !== null}
            onClick={() => fileInput.current?.click()}
          >
            {choice.custom_url ? "Replace your image" : "Upload your image"}
          </Button>
          {choice.custom_url ? (
            <Button
              variant="ghost"
              size="sm"
              disabled={busy !== null}
              onClick={() => setConfirmRemove(true)}
            >
              Remove your image
            </Button>
          ) : null}
        </div>
        <p className="text-sm text-fg-muted">
          JPG, PNG or WebP, up to 8 MB. Location and camera details are removed.
        </p>
        {uploadError ? (
          <p role="alert" className="text-sm text-error-text">
            {uploadError}
          </p>
        ) : null}
      </div>

      <WallpaperSliders userId={userId} prefs={prefs} />

      <Modal
        open={confirmRemove}
        onClose={() => setConfirmRemove(false)}
        dismissible={busy === null}
        title="Remove your image?"
        description={`Your ${modeLabel.toLowerCase()} mode image will be deleted. You can upload another one any time.`}
        footer={
          <>
            <Button
              variant="outline"
              disabled={busy !== null}
              onClick={() => setConfirmRemove(false)}
            >
              Cancel
            </Button>
            <Button variant="danger" loading={busy === "remove"} onClick={() => void remove()}>
              Remove image
            </Button>
          </>
        }
      />
    </div>
  );
}

/** One wallpaper option: a native radio (keyboard: arrows) shown as a thumbnail. */
function Tile({
  name,
  value,
  checked,
  label,
  onSelect,
  children,
}: {
  name: string;
  value: string;
  checked: boolean;
  label: string;
  onSelect: () => void;
  children: React.ReactNode;
}) {
  return (
    <label
      className="flex cursor-pointer flex-col gap-1.5"
      data-testid={`wallpaper-option-${value}`}
    >
      <input
        type="radio"
        name={name}
        value={value}
        checked={checked}
        onChange={onSelect}
        className="peer sr-only"
      />
      <span
        className={cn(
          "relative block aspect-video overflow-hidden rounded-md border border-border-strong",
          "transition-shadow duration-(--duration-fast) peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-ring",
          checked && "border-primary ring-2 ring-primary",
        )}
        aria-hidden="true"
      >
        {children}
      </span>
      <span className={cn("text-sm", checked ? "font-medium text-fg" : "text-fg-muted")}>
        {label}
      </span>
    </label>
  );
}

/**
 * A small page mock-up with the wallpaper behind it, drawn in the mode being edited (its own
 * data-theme), so the dark wallpaper can be checked while the app is light and vice versa.
 */
function Preview({
  mode,
  accent,
  choice,
  opacity,
  blur,
}: {
  mode: WallpaperMode;
  accent: string;
  choice: WallpaperChoice;
  opacity: number;
  blur: number;
}) {
  return (
    <div
      data-theme={mode}
      data-accent={accent}
      data-testid="wallpaper-preview"
      className="relative isolate h-40 overflow-hidden rounded-lg border border-border bg-canvas sm:h-48"
    >
      <WallpaperView choice={choice} opacity={opacity} blur={blur} position="absolute" />
      <div className="relative flex h-full flex-col gap-3 p-4" aria-hidden="true">
        <span className="h-3 w-32 rounded-full bg-fg" />
        <div className="grid flex-1 grid-cols-3 gap-3">
          <span className="rounded-md border border-border bg-surface shadow-xs" />
          <span className="rounded-md border border-border bg-surface shadow-xs" />
          <span className="rounded-md border border-border bg-surface shadow-xs" />
        </div>
      </div>
    </div>
  );
}

/** Slider values being edited, and the preferences to restore if saving them fails. */
interface SliderDraft {
  opacity: number;
  blur: number;
  before: Preferences;
}

function WallpaperSliders({ userId, prefs }: { userId: string; prefs: Preferences }) {
  const { toast } = useToast();
  const [draft, setDraft] = useState<SliderDraft | null>(null);
  // While nothing is being edited the sliders show the saved values (or another tab's).
  const opacity = draft?.opacity ?? prefs.wallpaper.opacity;
  const blur = draft?.blur ?? prefs.wallpaper.blur;

  function change(next: { opacity?: number; blur?: number }) {
    setDraft({ opacity, blur, ...next, before: draft?.before ?? prefs });
    previewWallpaper(userId, next);
  }

  // Save after a pause; every move restarts the wait.
  useEffect(() => {
    if (!draft) return;
    const timer = window.setTimeout(() => {
      saveWallpaper(userId, { opacity: draft.opacity, blur: draft.blur }, draft.before)
        .catch(() =>
          toast({
            variant: "error",
            title: "Wallpaper not saved",
            description: "Please try again.",
          }),
        )
        .finally(() => setDraft((d) => (d === draft ? null : d)));
    }, SLIDER_SAVE_DELAY_MS);
    return () => window.clearTimeout(timer);
  }, [draft, userId, toast]);

  const disabled =
    prefs.wallpaper.light.source === "none" && prefs.wallpaper.dark.source === "none";

  return (
    <div className="grid gap-5 sm:grid-cols-2">
      <Slider
        label="Visibility"
        hint="How strongly the wallpaper shows through."
        min={OPACITY_MIN}
        max={OPACITY_MAX}
        step={5}
        value={opacity}
        unit="%"
        disabled={disabled}
        onChange={(v) => change({ opacity: v })}
      />
      <Slider
        label="Blur"
        hint="Softens busy images."
        min={0}
        max={BLUR_MAX}
        step={1}
        value={blur}
        unit=" px"
        disabled={disabled}
        onChange={(v) => change({ blur: v })}
      />
    </div>
  );
}

function Slider({
  label,
  hint,
  min,
  max,
  step,
  value,
  unit,
  disabled,
  onChange,
}: {
  label: string;
  hint: string;
  min: number;
  max: number;
  step: number;
  value: number;
  unit: string;
  disabled?: boolean;
  onChange: (value: number) => void;
}) {
  const id = useId();
  return (
    <div className={cn("flex flex-col gap-2", disabled && "opacity-50")}>
      <div className="flex items-center justify-between gap-2">
        <label htmlFor={id} className="text-sm font-medium text-fg">
          {label}
        </label>
        <output htmlFor={id} className="text-sm text-fg-muted tabular-nums">
          {value}
          {unit}
        </output>
      </div>
      <input
        id={id}
        type="range"
        min={min}
        max={max}
        step={step}
        value={value}
        disabled={disabled}
        aria-describedby={`${id}-hint`}
        onChange={(e) => onChange(Number(e.target.value))}
        className="focus-ring h-2 w-full cursor-pointer accent-primary disabled:cursor-not-allowed"
      />
      <p id={`${id}-hint`} className="text-sm text-fg-muted">
        {hint}
      </p>
    </div>
  );
}
