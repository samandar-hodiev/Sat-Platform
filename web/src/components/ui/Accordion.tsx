"use client";

import { createContext, useContext, useId, useState, type ReactNode } from "react";

import { cn } from "@/lib/cn";

import { ChevronDownIcon } from "./icons";

interface AccordionContextValue {
  isOpen: (value: string) => boolean;
  toggle: (value: string) => void;
}

const AccordionContext = createContext<AccordionContextValue | null>(null);

export interface AccordionProps {
  /** "single": opening one item closes the others. */
  type?: "single" | "multiple";
  defaultValue?: string[];
  className?: string;
  children: ReactNode;
}

export function Accordion({
  type = "single",
  defaultValue = [],
  className,
  children,
}: AccordionProps) {
  const [open, setOpen] = useState<string[]>(defaultValue);
  const value: AccordionContextValue = {
    isOpen: (item) => open.includes(item),
    toggle: (item) =>
      setOpen((current) =>
        current.includes(item)
          ? current.filter((entry) => entry !== item)
          : type === "single"
            ? [item]
            : [...current, item],
      ),
  };
  return (
    <AccordionContext.Provider value={value}>
      <div
        className={cn(
          "divide-y divide-border rounded-lg border border-border bg-surface",
          className,
        )}
      >
        {children}
      </div>
    </AccordionContext.Provider>
  );
}

export interface AccordionItemProps {
  value: string;
  title: ReactNode;
  /** Extra content on the right of the header, e.g. a status Badge. */
  meta?: ReactNode;
  /** Heading level of the header for the page outline. */
  headingLevel?: 2 | 3 | 4;
  children: ReactNode;
}

export function AccordionItem({
  value,
  title,
  meta,
  headingLevel = 3,
  children,
}: AccordionItemProps) {
  const context = useContext(AccordionContext);
  if (!context) throw new Error("AccordionItem must be used inside <Accordion>.");
  const id = useId();
  const open = context.isOpen(value);
  const Heading = `h${headingLevel}` as const;

  return (
    <div>
      <Heading className="m-0">
        <button
          type="button"
          id={`${id}-trigger`}
          aria-expanded={open}
          aria-controls={`${id}-panel`}
          onClick={() => context.toggle(value)}
          className="focus-ring flex w-full items-center gap-3 rounded-lg px-4 py-3 text-left text-sm font-medium text-fg hover:bg-surface-hover"
        >
          <span className="flex-1">{title}</span>
          {meta}
          <ChevronDownIcon
            className={cn(
              "shrink-0 text-fg-subtle transition-transform duration-(--duration-base)",
              open && "rotate-180",
            )}
          />
        </button>
      </Heading>
      <div
        id={`${id}-panel`}
        role="region"
        aria-labelledby={`${id}-trigger`}
        hidden={!open}
        className="px-4 pb-4 text-sm text-fg-muted"
      >
        {children}
      </div>
    </div>
  );
}
