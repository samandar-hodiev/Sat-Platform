"use client";

import { useTheme } from "@/components/theme/ThemeProvider";
import { Switch } from "@/components/ui";
import { MonitorIcon, MoonIcon, SunIcon } from "@/components/ui/icons";
import type { Accent, Density, Motion, ThemeMode } from "@/lib/theme/theme";

import { useSidebarState, writeSidebarState } from "@/features/learner-shell";
import { RadioChips } from "@/features/onboarding/components/ChoiceChips";

import { Section } from "./Section";
import { WallpaperSettings } from "./WallpaperSettings";

const THEME_OPTIONS: { value: ThemeMode; label: React.ReactNode }[] = [
  { value: "light", label: <Labelled icon={<SunIcon />}>Light</Labelled> },
  { value: "dark", label: <Labelled icon={<MoonIcon />}>Dark</Labelled> },
  { value: "system", label: <Labelled icon={<MonitorIcon />}>System</Labelled> },
];

const ACCENT_LABELS: Record<Accent, string> = {
  neutral: "Neutral",
  blue: "Blue",
  indigo: "Indigo",
  violet: "Violet",
  teal: "Teal",
  rose: "Rose",
};

const DENSITY_OPTIONS: { value: Density; label: string }[] = [
  { value: "comfortable", label: "Comfortable" },
  { value: "compact", label: "Compact" },
];

const MOTION_OPTIONS: { value: Motion; label: string }[] = [
  { value: "system", label: "Device setting" },
  { value: "reduced", label: "Reduced" },
  { value: "full", label: "Full" },
];

function Labelled({ icon, children }: { icon: React.ReactNode; children: React.ReactNode }) {
  return (
    <span className="inline-flex items-center gap-2">
      {icon}
      {children}
    </span>
  );
}

/**
 * The accent's own colour: the swatch carries data-accent, so the accent tokens (globals.css)
 * resolve to that accent for this element only.
 */
function AccentSwatch({ accent }: { accent: Accent }) {
  return (
    <span className="inline-flex items-center gap-2">
      <span
        data-accent={accent}
        aria-hidden="true"
        className="size-4 shrink-0 rounded-full border border-border-strong bg-primary"
      />
      {ACCENT_LABELS[accent]}
    </span>
  );
}

/**
 * Settings → Appearance (PRD §4.4, §20). Every change applies at once on this device and is
 * saved to the account by PreferencesSync.
 */
export function AppearanceSettings({
  initialSidebar,
}: {
  initialSidebar: "expanded" | "collapsed";
}) {
  const theme = useTheme();
  const sidebar = useSidebarState(initialSidebar);

  return (
    <div className="flex flex-col gap-6">
      <Section
        title="Theme and color"
        description="Changes apply right away and follow your account to other devices."
      >
        <RadioChips<ThemeMode>
          legend="Theme"
          hint="System follows your device's light or dark setting."
          options={THEME_OPTIONS}
          value={theme.mode}
          onChange={theme.setMode}
          columns="grid-cols-1 sm:grid-cols-3"
        />
        <RadioChips<Accent>
          legend="Accent color"
          options={(Object.keys(ACCENT_LABELS) as Accent[]).map((a) => ({
            value: a,
            label: <AccentSwatch accent={a} />,
          }))}
          value={theme.accent}
          onChange={theme.setAccent}
          columns="grid-cols-2 sm:grid-cols-3 lg:grid-cols-6"
        />
      </Section>

      <WallpaperSettings />

      <Section title="Layout">
        <RadioChips<Density>
          legend="Density"
          hint="Compact fits more on the screen with smaller controls."
          options={DENSITY_OPTIONS}
          value={theme.density}
          onChange={theme.setDensity}
          columns="grid-cols-2"
        />
        <Switch
          label="Collapse the sidebar"
          description="Show only icons in the sidebar on large screens."
          checked={sidebar === "collapsed"}
          onCheckedChange={(on) => writeSidebarState(on ? "collapsed" : "expanded")}
        />
      </Section>

      <Section title="Motion">
        <RadioChips<Motion>
          legend="Animations"
          hint="Reduced turns off transitions. Device setting follows your system's reduce-motion option."
          options={MOTION_OPTIONS}
          value={theme.motion}
          onChange={theme.setMotion}
          columns="grid-cols-1 sm:grid-cols-3"
        />
      </Section>
    </div>
  );
}
