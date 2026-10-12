"use client";

import { useState, type ReactNode } from "react";

import { useShellAccount } from "../account";
import { writeSidebarState, type SidebarState } from "../sidebarState";
import { LearnerHeader } from "./LearnerHeader";
import { MobileDrawer } from "./MobileDrawer";
import { Sidebar } from "./Sidebar";

export interface LearnerShellProps {
  /** Read from the cookie on the server so the first paint has the right sidebar width. */
  initialSidebar: SidebarState;
  children: ReactNode;
}

/** Learner layout (PRD §10.1): collapsible sidebar, header, main content; drawer below lg. */
export function LearnerShell({ initialSidebar, children }: LearnerShellProps) {
  const account = useShellAccount();
  const [collapsed, setCollapsed] = useState(initialSidebar === "collapsed");
  const [drawerOpen, setDrawerOpen] = useState(false);

  const toggleSidebar = () => {
    const next = !collapsed;
    setCollapsed(next);
    writeSidebarState(next ? "collapsed" : "expanded");
  };

  return (
    <div className="flex min-h-dvh bg-canvas text-fg">
      <a
        href="#main"
        className="focus-ring sr-only z-(--z-tooltip) rounded-md bg-surface px-3 py-2 text-sm font-medium text-fg shadow-md focus:not-sr-only focus:fixed focus:top-3 focus:left-3"
      >
        Skip to content
      </a>
      <Sidebar collapsed={collapsed} onToggle={toggleSidebar} />
      <MobileDrawer open={drawerOpen} onClose={() => setDrawerOpen(false)} />
      <div className="flex min-w-0 flex-1 flex-col">
        <LearnerHeader
          account={account}
          drawerOpen={drawerOpen}
          onOpenDrawer={() => setDrawerOpen(true)}
        />
        <main id="main" tabIndex={-1} className="flex-1 px-4 py-6 outline-none sm:px-6 lg:px-8">
          <div className="mx-auto w-full max-w-6xl">{children}</div>
        </main>
      </div>
    </div>
  );
}
