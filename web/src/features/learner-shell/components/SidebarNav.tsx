"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useId } from "react";

import { cn } from "@/lib/cn";

import { LEARNER_NAV, activeNavItem } from "../nav";

export interface SidebarNavProps {
  /** Icons only; labels stay available to screen readers and as a hover title. */
  collapsed?: boolean;
  /** Called after a link is chosen, e.g. to close the mobile drawer. */
  onNavigate?: () => void;
}

/** The three PRD §10.2 groups (Learn, Track, Account) with the current page marked. */
export function SidebarNav({ collapsed = false, onNavigate }: SidebarNavProps) {
  const pathname = usePathname() ?? "";
  const active = activeNavItem(pathname);
  const id = useId();

  return (
    <nav aria-label="Main" className="flex flex-col gap-5">
      {LEARNER_NAV.map((group) => {
        const headingId = `${id}-${group.label}`;
        return (
          <div key={group.label} className="flex flex-col gap-1">
            <p
              id={headingId}
              className={cn(
                "px-3 text-xs font-medium tracking-wide text-fg-subtle uppercase",
                collapsed && "sr-only",
              )}
            >
              {group.label}
            </p>
            {collapsed ? <div aria-hidden className="mx-3 mb-1 h-px bg-border" /> : null}
            <ul aria-labelledby={headingId} className="flex flex-col gap-0.5">
              {group.items.map((item) => {
                const current = active?.href === item.href;
                const Icon = item.icon;
                return (
                  <li key={item.href}>
                    <Link
                      href={item.href}
                      onClick={onNavigate}
                      aria-current={current ? "page" : undefined}
                      title={collapsed ? item.label : undefined}
                      data-testid={`nav-${item.href}`}
                      className={cn(
                        "focus-ring flex h-(--control-md) items-center gap-3 rounded-md text-sm font-medium",
                        "transition-colors duration-(--duration-fast) ease-standard",
                        collapsed ? "justify-center px-0" : "px-3",
                        current
                          ? "bg-primary-soft text-primary-text"
                          : "text-fg-muted hover:bg-surface-hover hover:text-fg",
                      )}
                    >
                      <Icon className="shrink-0" />
                      <span className={cn("truncate", collapsed && "sr-only")}>{item.label}</span>
                    </Link>
                  </li>
                );
              })}
            </ul>
          </div>
        );
      })}
    </nav>
  );
}
