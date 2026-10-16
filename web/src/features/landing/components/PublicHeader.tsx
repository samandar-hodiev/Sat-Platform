"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";

import { ThemeSwitcher } from "@/components/theme/ThemeSwitcher";
import { IconButton, XIcon, buttonClasses } from "@/components/ui";

import { HEADER_NAV, PUBLIC_ROUTES } from "../content";
import { MenuIcon } from "../icons";
import { PublicBrand } from "./PublicBrand";

const navLink =
  "focus-ring rounded-md px-3 py-2 text-sm font-medium whitespace-nowrap text-fg-muted transition-colors " +
  "duration-(--duration-fast) hover:bg-surface-hover hover:text-fg";

/** Public site header (PRD §5.1): logo, section links, Sign In, Create Account, mobile menu. */
export function PublicHeader() {
  const [menuOpen, setMenuOpen] = useState(false);

  return (
    <header className="sticky top-0 z-(--z-header) border-b border-border bg-canvas">
      <div className="mx-auto flex h-16 max-w-6xl items-center gap-4 px-4 sm:px-6">
        <PublicBrand />
        <nav aria-label="Main" className="hidden flex-1 justify-center xl:flex">
          <ul className="flex items-center gap-1">
            {HEADER_NAV.map((link) => (
              <li key={link.href}>
                <Link href={link.href} className={navLink}>
                  {link.label}
                </Link>
              </li>
            ))}
          </ul>
        </nav>
        <div className="ml-auto flex items-center gap-2 xl:ml-0">
          {/* Wrappers carry the responsive visibility: `hidden` on the components themselves
              would lose to their own inline-flex. */}
          <div className="hidden xl:block">
            <ThemeSwitcher />
          </div>
          <div className="hidden items-center gap-2 sm:flex">
            <Link
              href={PUBLIC_ROUTES.signIn}
              className={buttonClasses({ variant: "ghost" })}
              data-testid="header-sign-in"
            >
              Sign In
            </Link>
            <Link
              href={PUBLIC_ROUTES.signUp}
              className={buttonClasses()}
              data-testid="header-sign-up"
            >
              Create Account
            </Link>
          </div>
          <IconButton
            aria-label="Open menu"
            aria-haspopup="dialog"
            aria-expanded={menuOpen}
            onClick={() => setMenuOpen(true)}
            className="xl:hidden"
            data-testid="public-menu-button"
          >
            <MenuIcon size={18} />
          </IconButton>
        </div>
      </div>
      <PublicMobileMenu open={menuOpen} onClose={() => setMenuOpen(false)} />
    </header>
  );
}

/**
 * Menu below xl on the native <dialog>, like the learner MobileDrawer: showModal() gives the
 * top layer, inert background, focus containment, Escape and focus return. Links close it.
 */
function PublicMobileMenu({ open, onClose }: { open: boolean; onClose: () => void }) {
  const ref = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const dialog = ref.current;
    if (!dialog) return;
    if (open && !dialog.open) {
      dialog.showModal();
      const previousOverflow = document.body.style.overflow;
      document.body.style.overflow = "hidden";
      return () => {
        document.body.style.overflow = previousOverflow;
      };
    }
    if (!open && dialog.open) dialog.close();
  }, [open]);

  return (
    <dialog
      ref={ref}
      aria-label="Menu"
      data-testid="public-menu"
      onCancel={(event) => {
        event.preventDefault();
        onClose();
      }}
      onClose={() => {
        if (open) onClose();
      }}
      onClick={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
      className="m-0 ml-auto h-dvh max-h-dvh w-80 max-w-[85vw] border-l border-border bg-surface p-0 text-fg shadow-lg backdrop:animate-fade-in open:animate-fade-in xl:hidden"
    >
      {open ? (
        <div className="flex h-full flex-col">
          <div className="flex h-16 shrink-0 items-center justify-between pr-2 pl-5">
            <PublicBrand onNavigate={onClose} />
            <IconButton aria-label="Close menu" onClick={onClose} autoFocus>
              <XIcon size={18} />
            </IconButton>
          </div>
          <nav aria-label="Main" className="flex-1 overflow-y-auto px-3 py-3">
            <ul className="flex flex-col gap-1">
              {HEADER_NAV.map((link) => (
                <li key={link.href}>
                  <Link
                    href={link.href}
                    onClick={onClose}
                    className="focus-ring flex rounded-md px-3 py-2.5 text-base font-medium text-fg hover:bg-surface-hover"
                  >
                    {link.label}
                  </Link>
                </li>
              ))}
            </ul>
          </nav>
          <div className="flex flex-col gap-3 border-t border-border p-4">
            <Link
              href={PUBLIC_ROUTES.signUp}
              onClick={onClose}
              className={buttonClasses({ fullWidth: true })}
            >
              Create Account
            </Link>
            <Link
              href={PUBLIC_ROUTES.signIn}
              onClick={onClose}
              className={buttonClasses({ variant: "outline", fullWidth: true })}
            >
              Sign In
            </Link>
            <ThemeSwitcher showLabels className="self-center" />
          </div>
        </div>
      ) : null}
    </dialog>
  );
}
