"use client";

import { useEffect, useRef } from "react";

import { IconButton, XIcon } from "@/components/ui";

import { Brand } from "./Brand";
import { SidebarNav } from "./SidebarNav";

export interface MobileDrawerProps {
  open: boolean;
  onClose: () => void;
}

/**
 * Navigation drawer below lg. Built on the native <dialog> (like the UI kit Modal): showModal()
 * gives the top layer, an inert background, focus containment, Escape and focus return to the
 * menu button. Focus starts on the close button; choosing a link closes the drawer.
 */
export function MobileDrawer({ open, onClose }: MobileDrawerProps) {
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
      aria-label="Navigation"
      data-testid="learner-drawer"
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
      className="m-0 h-dvh max-h-dvh w-72 max-w-[85vw] border-r border-border bg-surface p-0 text-fg shadow-lg backdrop:animate-fade-in open:animate-fade-in lg:hidden"
    >
      {open ? (
        <div className="flex h-full flex-col">
          <div className="flex h-14 shrink-0 items-center justify-between pr-2 pl-5">
            <Brand onNavigate={onClose} />
            <IconButton aria-label="Close navigation" onClick={onClose} autoFocus>
              <XIcon size={18} />
            </IconButton>
          </div>
          <div className="flex-1 overflow-y-auto px-3 py-3">
            <SidebarNav onNavigate={onClose} />
          </div>
        </div>
      ) : null}
    </dialog>
  );
}
