"use client";

import {
  cloneElement,
  isValidElement,
  useEffect,
  useId,
  useRef,
  useState,
  type ReactElement,
  type ReactNode,
} from "react";

import { cn } from "@/lib/cn";

export interface TooltipProps {
  content: ReactNode;
  side?: "top" | "bottom";
  /** Hover delay; keyboard focus shows it immediately. */
  delay?: number;
  /** A single focusable element, e.g. an IconButton. */
  children: ReactElement<{ "aria-describedby"?: string }>;
}

/**
 * Short supplementary text on hover and keyboard focus (role="tooltip", linked with
 * aria-describedby). Escape hides it. Never put essential information or interactive
 * content in a tooltip — touch users cannot hover.
 */
export function Tooltip({ content, side = "top", delay = 300, children }: TooltipProps) {
  const id = useId();
  const [open, setOpen] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  const show = (wait: number) => {
    clearTimeout(timer.current);
    timer.current = setTimeout(() => setOpen(true), wait);
  };
  const hide = () => {
    clearTimeout(timer.current);
    setOpen(false);
  };

  useEffect(() => () => clearTimeout(timer.current), []);

  useEffect(() => {
    if (!open) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") hide();
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [open]);

  if (!isValidElement(children)) return children;
  const describedBy = [children.props["aria-describedby"], open ? id : null]
    .filter(Boolean)
    .join(" ");

  return (
    <span
      className="relative inline-flex"
      onPointerEnter={() => show(delay)}
      onPointerLeave={hide}
      onFocus={() => show(0)}
      onBlur={hide}
    >
      {cloneElement(children, { "aria-describedby": describedBy || undefined })}
      {open ? (
        <span
          role="tooltip"
          id={id}
          className={cn(
            "pointer-events-none absolute left-1/2 z-(--z-tooltip) w-max max-w-64 -translate-x-1/2 animate-fade-in",
            "rounded-md bg-fg px-2 py-1 text-xs font-medium text-canvas shadow-md",
            side === "top" ? "bottom-full mb-1.5" : "top-full mt-1.5",
          )}
        >
          {content}
        </span>
      ) : null}
    </span>
  );
}
