"use client";

import { useEffect, useEffectEvent, useRef } from "react";

import { useTheme } from "@/components/theme/ThemeProvider";
import { useToast } from "@/components/ui";

import { useSidebarState, writeSidebarState } from "@/features/learner-shell";

import { updatePreferences, type Preferences } from "../api";
import { setPreferencesData, usePreferences } from "../preferencesStore";

/** The appearance values kept both in the device cookies and in the account. */
interface Appearance {
  theme: Preferences["theme"];
  accent: Preferences["accent"];
  density: Preferences["density"];
  motion: Preferences["motion"];
  sidebar_collapsed: boolean;
}

function fromAccount(p: Preferences): Appearance {
  return {
    theme: p.theme,
    accent: p.accent,
    density: p.density,
    motion: p.motion,
    sidebar_collapsed: p.sidebar_collapsed,
  };
}

const same = (a: Appearance, b: Appearance) =>
  a.theme === b.theme &&
  a.accent === b.accent &&
  a.density === b.density &&
  a.motion === b.motion &&
  a.sidebar_collapsed === b.sidebar_collapsed;

/** Waits for a burst of changes (e.g. arrow keys through the accents) before saving. */
export const SYNC_DELAY_MS = 500;

/**
 * Keeps the device appearance (theme cookie, sidebar cookie) and the account in step
 * (PRD §4.6). Decided for 2.11: the account wins. On the first load the stored account values
 * are applied to this device; an account that never saved anything takes this device's values
 * instead. After that every change made here (Settings, sidebar toggle) is saved to the account.
 * A failed save keeps the change on this device and says so; the next change sends everything.
 */
export function PreferencesSync({ initialSidebar }: { initialSidebar: "expanded" | "collapsed" }) {
  const { state } = usePreferences();
  const theme = useTheme();
  const sidebar = useSidebarState(initialSidebar);
  const { toast } = useToast();

  const local: Appearance = {
    theme: theme.mode,
    accent: theme.accent,
    density: theme.density,
    motion: theme.motion,
    sidebar_collapsed: sidebar === "collapsed",
  };
  // The device values at the moment an effect runs, without making them an effect dependency.
  const deviceAppearance = useEffectEvent(() => local);
  // What the account holds, as far as this tab knows; null until the first sync.
  const synced = useRef<{ userId: string; appearance: Appearance } | null>(null);

  const userId = state.status === "ready" ? state.userId : null;
  const account = state.status === "ready" ? state.preferences : null;

  // First sync for this learner.
  useEffect(() => {
    if (!userId || !account || synced.current?.userId === userId) return;
    if (account.stored) {
      const a = fromAccount(account);
      synced.current = { userId, appearance: a };
      theme.setPreferences({
        mode: a.theme,
        accent: a.accent,
        density: a.density,
        motion: a.motion,
      });
      writeSidebarState(a.sidebar_collapsed ? "collapsed" : "expanded");
      return;
    }
    const device = deviceAppearance();
    synced.current = { userId, appearance: device };
    updatePreferences(device)
      .then((next) => setPreferencesData(userId, next))
      .catch(() => {
        // Retried with the next change.
        if (synced.current?.userId === userId)
          synced.current = { userId, appearance: fromAccount(account) };
      });
  }, [userId, account, theme]);

  // Later changes made on this device.
  const key = JSON.stringify(local);
  useEffect(() => {
    const s = synced.current;
    const sending = deviceAppearance();
    if (!userId || !s || s.userId !== userId || same(sending, s.appearance)) return;
    // Every change restarts the timer, so what is sent is the latest state.
    const timer = window.setTimeout(async () => {
      try {
        const next = await updatePreferences(sending);
        if (synced.current?.userId !== userId) return;
        synced.current = { userId, appearance: fromAccount(next) };
        setPreferencesData(userId, next);
      } catch {
        toast({
          variant: "error",
          title: "Appearance not saved to your account",
          description: "It is applied on this device. We'll try again with your next change.",
        });
      }
    }, SYNC_DELAY_MS);
    return () => window.clearTimeout(timer);
  }, [key, userId, toast]);

  return null;
}
