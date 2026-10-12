"use client";

import type { ReactNode } from "react";

import { cn } from "@/lib/cn";
import type { ThemeMode } from "@/lib/theme/theme";

import { MonitorIcon, MoonIcon, SunIcon } from "../ui/icons";
import { useTheme } from "./ThemeProvider";

const options: Array<{ mode: ThemeMode; label: string; icon: ReactNode }> = [
  { mode: "light", label: "Light", icon: <SunIcon /> },
  { mode: "dark", label: "Dark", icon: <MoonIcon /> },
  { mode: "system", label: "System", icon: <MonitorIcon /> },
];

/**
 * Compact Light / Dark / System control for headers. The full Appearance page (accent, density,
 * motion, wallpaper) belongs to Settings (session 2.11).
 */
export function ThemeSwitcher({
  showLabels = false,
  className,
}: {
  showLabels?: boolean;
  className?: string;
}) {
  const { mode, setMode } = useTheme();
  return (
    <div
      role="radiogroup"
      aria-label="Color theme"
      className={cn(
        "inline-flex gap-0.5 rounded-lg border border-border bg-surface-muted p-0.5",
        className,
      )}
    >
      {options.map((option) => {
        const selected = mode === option.mode;
        return (
          <button
            key={option.mode}
            type="button"
            role="radio"
            aria-checked={selected}
            aria-label={showLabels ? undefined : option.label}
            title={showLabels ? undefined : option.label}
            onClick={() => setMode(option.mode)}
            className={cn(
              "focus-ring inline-flex h-7 items-center gap-1.5 rounded-md px-2 text-xs font-medium",
              "transition-colors duration-(--duration-fast)",
              selected ? "bg-surface text-fg shadow-xs" : "text-fg-muted hover:text-fg",
            )}
          >
            {option.icon}
            {showLabels ? option.label : null}
          </button>
        );
      })}
    </div>
  );
}
