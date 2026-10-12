"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

import {
  Avatar,
  ChevronDownIcon,
  DropdownMenu,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  Skeleton,
  useToast,
} from "@/components/ui";
import { cn } from "@/lib/cn";

import type { ShellAccount } from "../account";
import { CreditCardIcon, LogOutIcon, SettingsIcon, UserIcon } from "../icons";
import { findNavItem } from "../nav";

const ACCOUNT_LINKS = [
  { href: "/app/profile", icon: UserIcon },
  { href: "/app/settings", icon: SettingsIcon },
  { href: "/app/subscription", icon: CreditCardIcon },
] as const;

/**
 * Avatar + profile dropdown (PRD §10.3). Shows only what the session really knows: a skeleton
 * while loading, a neutral icon and a disabled "Sign out" when there is no session data.
 */
export function UserMenu({ account }: { account: ShellAccount }) {
  const router = useRouter();
  const { toast } = useToast();
  const [signingOut, setSigningOut] = useState(false);

  if (account.status === "loading") {
    return (
      <span role="status" aria-label="Loading account" className="inline-flex">
        <Skeleton className="size-9 rounded-full" />
      </span>
    );
  }

  const user = account.status === "ready" ? account.user : null;

  const signOut = async () => {
    if (account.status !== "ready" || signingOut) return;
    setSigningOut(true);
    try {
      await account.signOut();
    } catch {
      toast({ variant: "error", title: "Could not sign out", description: "Please try again." });
    } finally {
      setSigningOut(false);
    }
  };

  return (
    <DropdownMenu
      align="end"
      className="w-60"
      trigger={({ ref, ...props }) => (
        <button
          ref={ref}
          type="button"
          aria-label="Account menu"
          data-testid="user-menu-trigger"
          className={cn(
            "focus-ring inline-flex items-center gap-1 rounded-full p-0.5 text-fg-muted",
            "transition-colors duration-(--duration-fast) hover:text-fg",
          )}
          {...props}
        >
          {user ? (
            <Avatar name={user.name} src={user.avatarUrl} />
          ) : (
            <span className="inline-flex size-9 items-center justify-center rounded-full bg-surface-muted">
              <UserIcon />
            </span>
          )}
          <ChevronDownIcon className="hidden sm:block" />
        </button>
      )}
    >
      {user ? (
        <div className="px-2 py-1.5">
          <p className="truncate text-sm font-medium text-fg">{user.name}</p>
          <p className="truncate text-xs text-fg-muted">{user.email}</p>
        </div>
      ) : (
        <DropdownMenuLabel>Account details are unavailable</DropdownMenuLabel>
      )}
      <DropdownMenuSeparator />
      {ACCOUNT_LINKS.map(({ href, icon: Icon }) => (
        <DropdownMenuItem key={href} icon={<Icon size={16} />} onSelect={() => router.push(href)}>
          {findNavItem(href)?.label}
        </DropdownMenuItem>
      ))}
      <DropdownMenuSeparator />
      <DropdownMenuItem
        icon={<LogOutIcon size={16} />}
        disabled={!user || signingOut}
        onSelect={signOut}
      >
        {signingOut ? "Signing out…" : "Sign out"}
      </DropdownMenuItem>
    </DropdownMenu>
  );
}
