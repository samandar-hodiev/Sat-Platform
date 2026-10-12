"use client";

import { useEffect, useId, useRef, type ReactNode } from "react";

import { cn } from "@/lib/cn";

import { IconButton } from "./IconButton";
import { XIcon } from "./icons";

const sizes = { sm: "max-w-sm", md: "max-w-lg", lg: "max-w-2xl" } as const;

export interface ModalProps {
  open: boolean;
  /** Called on Escape, backdrop click and the close button. */
  onClose: () => void;
  title: ReactNode;
  description?: ReactNode;
  /** Action buttons, right-aligned at the bottom. */
  footer?: ReactNode;
  size?: keyof typeof sizes;
  /** Disable Escape/backdrop closing, e.g. while a submit is in flight. */
  dismissible?: boolean;
  className?: string;
  children?: ReactNode;
}

/**
 * Built on the native <dialog>: showModal() gives the top layer, inert background, focus
 * containment, Escape handling and focus return to the opener without extra code.
 */
export function Modal({
  open,
  onClose,
  title,
  description,
  footer,
  size = "md",
  dismissible = true,
  className,
  children,
}: ModalProps) {
  const ref = useRef<HTMLDialogElement>(null);
  const id = useId();

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
      aria-labelledby={`${id}-title`}
      aria-describedby={description ? `${id}-description` : undefined}
      onCancel={(event) => {
        event.preventDefault();
        if (dismissible) onClose();
      }}
      onClick={(event) => {
        if (dismissible && event.target === event.currentTarget) onClose();
      }}
      className={cn(
        "m-auto w-[calc(100%-2rem)] rounded-xl border border-border bg-surface p-0 text-fg shadow-lg",
        "backdrop:animate-fade-in open:animate-pop-in",
        sizes[size],
        className,
      )}
    >
      {open ? (
        <div className="flex max-h-[85dvh] flex-col">
          <div className="flex items-start gap-4 px-6 pt-5 pb-3">
            <div className="flex flex-1 flex-col gap-1">
              <h2 id={`${id}-title`} className="text-lg font-semibold text-fg">
                {title}
              </h2>
              {description ? (
                <p id={`${id}-description`} className="text-sm text-fg-muted">
                  {description}
                </p>
              ) : null}
            </div>
            {dismissible ? (
              <IconButton aria-label="Close" size="sm" onClick={onClose} className="-mt-1 -mr-2">
                <XIcon />
              </IconButton>
            ) : null}
          </div>
          {children ? <div className="overflow-y-auto px-6 py-2 text-sm">{children}</div> : null}
          {footer ? (
            <div className="flex flex-col-reverse gap-2 px-6 pt-3 pb-5 sm:flex-row sm:justify-end">
              {footer}
            </div>
          ) : null}
        </div>
      ) : null}
    </dialog>
  );
}
