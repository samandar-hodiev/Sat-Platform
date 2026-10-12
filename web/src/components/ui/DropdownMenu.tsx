"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useId,
  useRef,
  useState,
  type KeyboardEvent,
  type ReactNode,
} from "react";

import { cn } from "@/lib/cn";

interface MenuContextValue {
  close: (returnFocus?: boolean) => void;
}

const MenuContext = createContext<MenuContextValue | null>(null);

export interface DropdownTriggerProps {
  id: string;
  "aria-haspopup": "menu";
  "aria-expanded": boolean;
  "aria-controls": string | undefined;
  onClick: () => void;
  onKeyDown: (event: KeyboardEvent<HTMLElement>) => void;
}

export interface DropdownMenuProps {
  /** Render the trigger and spread the given props on it, e.g. `(p) => <IconButton {...p} … />`. */
  trigger: (props: DropdownTriggerProps & { ref: React.Ref<HTMLButtonElement> }) => ReactNode;
  align?: "start" | "end";
  className?: string;
  children: ReactNode;
}

/**
 * WAI-ARIA menu button: Enter/Space/ArrowDown open and focus the first item, arrows/Home/End
 * move, Escape closes and returns focus to the trigger, Tab or a click outside closes.
 * Positioned under the trigger; avoid placing it inside an `overflow-hidden` ancestor.
 */
export function DropdownMenu({ trigger, align = "start", className, children }: DropdownMenuProps) {
  const id = useId();
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  const focusOnOpen = useRef<"first" | "last">("first");

  const items = () =>
    Array.from(
      menuRef.current?.querySelectorAll<HTMLElement>(
        '[role="menuitem"]:not([aria-disabled="true"])',
      ) ?? [],
    );

  const close = useCallback((returnFocus = true) => {
    setOpen(false);
    if (returnFocus) triggerRef.current?.focus();
  }, []);

  useEffect(() => {
    if (!open) return;
    const list = items();
    (focusOnOpen.current === "last" ? list[list.length - 1] : list[0])?.focus();

    const onPointerDown = (event: PointerEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) close(false);
    };
    document.addEventListener("pointerdown", onPointerDown);
    return () => document.removeEventListener("pointerdown", onPointerDown);
  }, [open, close]);

  const openWith = (focus: "first" | "last") => {
    focusOnOpen.current = focus;
    setOpen(true);
  };

  const onTriggerKeyDown = (event: KeyboardEvent<HTMLElement>) => {
    if (event.key === "ArrowDown" || event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      openWith("first");
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      openWith("last");
    }
  };

  const onMenuKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    const list = items();
    const index = list.indexOf(document.activeElement as HTMLElement);
    const targets: Record<string, number> = {
      ArrowDown: (index + 1) % list.length,
      ArrowUp: (index - 1 + list.length) % list.length,
      Home: 0,
      End: list.length - 1,
    };
    if (event.key === "Escape") {
      event.preventDefault();
      close();
    } else if (event.key === "Tab") {
      close(false);
    } else if (targets[event.key] !== undefined) {
      event.preventDefault();
      list[targets[event.key] as number]?.focus();
    }
  };

  return (
    <div ref={rootRef} className="relative inline-flex">
      {trigger({
        ref: triggerRef,
        id: `${id}-trigger`,
        "aria-haspopup": "menu",
        "aria-expanded": open,
        "aria-controls": open ? `${id}-menu` : undefined,
        onClick: () => (open ? close(false) : openWith("first")),
        onKeyDown: onTriggerKeyDown,
      })}
      {open ? (
        <MenuContext.Provider value={{ close }}>
          <div
            ref={menuRef}
            id={`${id}-menu`}
            role="menu"
            aria-labelledby={`${id}-trigger`}
            onKeyDown={onMenuKeyDown}
            className={cn(
              "absolute top-full z-(--z-dropdown) mt-1 min-w-44 animate-pop-in rounded-lg border border-border bg-surface p-1 shadow-md",
              align === "end" ? "right-0" : "left-0",
              className,
            )}
          >
            {children}
          </div>
        </MenuContext.Provider>
      ) : null}
    </div>
  );
}

export interface DropdownMenuItemProps {
  onSelect?: () => void;
  disabled?: boolean;
  /** Destructive actions are shown in the error color. */
  destructive?: boolean;
  icon?: ReactNode;
  children: ReactNode;
}

export function DropdownMenuItem({
  onSelect,
  disabled,
  destructive,
  icon,
  children,
}: DropdownMenuItemProps) {
  const menu = useContext(MenuContext);
  const activate = () => {
    if (disabled) return;
    menu?.close();
    onSelect?.();
  };
  return (
    <div
      role="menuitem"
      tabIndex={-1}
      aria-disabled={disabled || undefined}
      onClick={activate}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          activate();
        }
      }}
      className={cn(
        "flex h-(--control-sm) cursor-pointer items-center gap-2 rounded-md px-2 text-sm outline-none select-none",
        "focus:bg-surface-hover hover:bg-surface-hover",
        destructive ? "text-error-text" : "text-fg",
        disabled && "cursor-not-allowed opacity-50",
      )}
    >
      {icon ? <span className="text-fg-subtle">{icon}</span> : null}
      {children}
    </div>
  );
}

export function DropdownMenuSeparator() {
  return <div role="separator" className="-mx-1 my-1 h-px bg-border" />;
}

export function DropdownMenuLabel({ children }: { children: ReactNode }) {
  return <div className="px-2 py-1.5 text-xs font-medium text-fg-subtle">{children}</div>;
}
