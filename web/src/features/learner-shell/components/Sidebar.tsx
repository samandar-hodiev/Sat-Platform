"use client";

import { cn } from "@/lib/cn";

import { PanelLeftIcon } from "../icons";
import { Brand } from "./Brand";
import { SidebarNav } from "./SidebarNav";

export interface SidebarProps {
  collapsed: boolean;
  onToggle: () => void;
}

/** Desktop sidebar (lg and up). Below lg the same navigation lives in the MobileDrawer. */
export function Sidebar({ collapsed, onToggle }: SidebarProps) {
  return (
    <aside
      data-testid="learner-sidebar"
      data-state={collapsed ? "collapsed" : "expanded"}
      className={cn(
        "sticky top-0 hidden h-dvh shrink-0 flex-col border-r border-border bg-surface lg:flex",
        "transition-[width] duration-(--duration-base) ease-standard",
        collapsed ? "w-16" : "w-64",
      )}
    >
      <div className={cn("flex h-14 shrink-0 items-center", collapsed ? "justify-center" : "px-5")}>
        <Brand compact={collapsed} />
      </div>
      <div className={cn("flex-1 overflow-y-auto py-3", collapsed ? "px-2" : "px-3")}>
        <SidebarNav collapsed={collapsed} />
      </div>
      <div className={cn("shrink-0 border-t border-border py-3", collapsed ? "px-2" : "px-3")}>
        <button
          type="button"
          onClick={onToggle}
          aria-expanded={!collapsed}
          aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
          title={collapsed ? "Expand sidebar" : undefined}
          className={cn(
            "focus-ring flex h-(--control-md) w-full items-center gap-3 rounded-md text-sm font-medium text-fg-muted",
            "transition-colors duration-(--duration-fast) ease-standard hover:bg-surface-hover hover:text-fg",
            collapsed ? "justify-center" : "px-3",
          )}
        >
          <PanelLeftIcon className="shrink-0" />
          {collapsed ? null : <span>Collapse</span>}
        </button>
      </div>
    </aside>
  );
}
