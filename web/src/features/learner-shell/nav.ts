// Learner navigation (PRD §10.2). Paths follow docs/ARCHITECTURE.md (learner app under /app).
// Pages are built by their sessions; until then a link opens the in-shell "Page not found".

import type { ComponentType } from "react";

import {
  BookIcon,
  BookmarkIcon,
  CalendarIcon,
  ChartIcon,
  ClipboardIcon,
  CreditCardIcon,
  FormulaIcon,
  HomeIcon,
  NotebookIcon,
  SettingsIcon,
  SparklesIcon,
  UserIcon,
  type ShellIconProps,
} from "./icons";

export interface NavItem {
  label: string;
  href: string;
  icon: ComponentType<ShellIconProps>;
  /** Session that builds the page, for whoever wires it up. */
  session: string;
}

export interface NavGroup {
  label: string;
  items: NavItem[];
}

export const LEARNER_HOME = "/app";

export const LEARNER_NAV: NavGroup[] = [
  {
    label: "Learn",
    items: [
      { label: "Dashboard", href: LEARNER_HOME, icon: HomeIcon, session: "2.04" },
      { label: "My Study Plan", href: "/app/plan", icon: CalendarIcon, session: "2.05" },
      { label: "Learn & Practice", href: "/app/learn", icon: BookIcon, session: "3.02" },
      { label: "Mock Exams", href: "/app/exams", icon: ClipboardIcon, session: "3.08" },
      {
        label: "Formula & Concept Library",
        href: "/app/library",
        icon: FormulaIcon,
        session: "3.10",
      },
      { label: "AI Study Assistant", href: "/app/assistant", icon: SparklesIcon, session: "2.06" },
    ],
  },
  {
    label: "Track",
    items: [
      { label: "My Progress", href: "/app/progress", icon: ChartIcon, session: "2.07" },
      { label: "Saved Questions", href: "/app/saved", icon: BookmarkIcon, session: "2.08" },
      { label: "Mistake Notebook", href: "/app/mistakes", icon: NotebookIcon, session: "2.09" },
    ],
  },
  {
    label: "Account",
    items: [
      { label: "My Profile", href: "/app/profile", icon: UserIcon, session: "2.10" },
      { label: "Settings", href: "/app/settings", icon: SettingsIcon, session: "2.11" },
      {
        label: "Subscription",
        href: "/app/subscription",
        icon: CreditCardIcon,
        session: "3.13",
      },
    ],
  },
];

const ALL_ITEMS = LEARNER_NAV.flatMap((group) => group.items);

export function findNavItem(href: string): NavItem | undefined {
  return ALL_ITEMS.find((item) => item.href === href);
}

/**
 * The item a path belongs to: an exact match, or the longest item path that prefixes it
 * (`/app/exams/123` → Mock Exams). Dashboard (`/app`) only matches itself, otherwise it would
 * claim every page.
 */
export function activeNavItem(pathname: string): NavItem | undefined {
  const path = pathname.length > 1 ? pathname.replace(/\/+$/, "") : pathname;
  let best: NavItem | undefined;
  for (const item of ALL_ITEMS) {
    const matches =
      path === item.href || (item.href !== LEARNER_HOME && path.startsWith(`${item.href}/`));
    if (matches && (!best || item.href.length > best.href.length)) best = item;
  }
  return best;
}
