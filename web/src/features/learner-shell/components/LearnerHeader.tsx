"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

import { IconButton } from "@/components/ui";

import type { ShellAccount } from "../account";
import { MenuIcon } from "../icons";
import { activeNavItem } from "../nav";
import { UserMenu } from "./UserMenu";

export interface LearnerHeaderProps {
  account: ShellAccount;
  drawerOpen: boolean;
  onOpenDrawer: () => void;
}

/**
 * Top bar of every learner page (PRD §10.3): menu button (below lg), the current section as a
 * breadcrumb, then actions and the profile menu. Pages render their own <h1>.
 *
 * Header actions are added by their sessions in the marked slot: notifications (3.14) and the
 * streak / progress indicator (2.04, 2.07). They are not shown before real data exists.
 */
export function LearnerHeader({ account, drawerOpen, onOpenDrawer }: LearnerHeaderProps) {
  const pathname = usePathname() ?? "";
  const section = activeNavItem(pathname);
  const isSectionRoot = section ? pathname.replace(/\/+$/, "") === section.href : false;

  return (
    <header className="sticky top-0 z-(--z-header) flex h-14 shrink-0 items-center gap-2 border-b border-border bg-canvas px-2 sm:px-4 lg:px-6">
      <IconButton
        aria-label="Open navigation"
        aria-expanded={drawerOpen}
        aria-haspopup="dialog"
        onClick={onOpenDrawer}
        className="lg:hidden"
        data-testid="open-drawer"
      >
        <MenuIcon />
      </IconButton>

      <nav aria-label="Breadcrumb" className="min-w-0 flex-1">
        {section ? (
          <ol className="flex min-w-0 items-center text-sm">
            <li className="min-w-0 truncate">
              {isSectionRoot ? (
                <span aria-current="page" className="font-medium text-fg">
                  {section.label}
                </span>
              ) : (
                <Link
                  href={section.href}
                  className="focus-ring rounded-sm font-medium text-fg-muted hover:text-fg"
                >
                  {section.label}
                </Link>
              )}
            </li>
          </ol>
        ) : null}
      </nav>

      <div className="flex shrink-0 items-center gap-1 sm:gap-2">
        {/* Header actions slot: notifications (3.14), streak / progress (2.04, 2.07). */}
        <UserMenu account={account} />
      </div>
    </header>
  );
}
