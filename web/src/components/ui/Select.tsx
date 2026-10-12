"use client";

import type { ComponentProps } from "react";

import { cn } from "@/lib/cn";

import { controlClasses } from "./controlStyles";
import { useFieldControl } from "./Field";
import { ChevronDownIcon } from "./icons";

export type SelectProps = ComponentProps<"select"> & { placeholder?: string };

/**
 * Native <select>: keyboard, screen reader and mobile pickers work out of the box.
 * `placeholder` adds a disabled empty first option.
 */
export function Select({ className, placeholder, children, ...props }: SelectProps) {
  const { invalid, ...field } = useFieldControl(props);
  return (
    <div className={cn("relative", className)}>
      <select
        {...props}
        {...field}
        className={controlClasses(invalid, "h-(--control-md) appearance-none pr-9 pl-3")}
      >
        {placeholder !== undefined ? (
          <option value="" disabled>
            {placeholder}
          </option>
        ) : null}
        {children}
      </select>
      <ChevronDownIcon className="pointer-events-none absolute top-1/2 right-3 -translate-y-1/2 text-fg-subtle" />
    </div>
  );
}
