"use client";

import { useId, useState, type ReactNode } from "react";

import { cn } from "@/lib/cn";

export interface SwitchProps {
  checked?: boolean;
  defaultChecked?: boolean;
  onCheckedChange?: (checked: boolean) => void;
  label?: ReactNode;
  description?: ReactNode;
  disabled?: boolean;
  id?: string;
  className?: string;
  "aria-label"?: string;
}

/** On/off setting that applies immediately (role="switch"). For form opt-ins use Checkbox. */
export function Switch({
  checked,
  defaultChecked = false,
  onCheckedChange,
  label,
  description,
  disabled,
  id,
  className,
  "aria-label": ariaLabel,
}: SwitchProps) {
  const generated = useId();
  const controlId = id ?? generated;
  const [internal, setInternal] = useState(defaultChecked);
  const isOn = checked ?? internal;

  const toggle = () => {
    if (checked === undefined) setInternal(!isOn);
    onCheckedChange?.(!isOn);
  };

  const control = (
    <button
      id={controlId}
      type="button"
      role="switch"
      aria-checked={isOn}
      aria-label={ariaLabel}
      aria-describedby={description ? `${controlId}-description` : undefined}
      disabled={disabled}
      onClick={toggle}
      className={cn(
        "focus-ring relative inline-flex h-5 w-9 shrink-0 items-center rounded-full",
        "transition-colors duration-(--duration-base) ease-standard disabled:cursor-not-allowed disabled:opacity-50",
        isOn ? "bg-primary" : "bg-border-strong",
      )}
    >
      <span
        aria-hidden="true"
        className={cn(
          "inline-block size-4 rounded-full bg-surface shadow-sm transition-transform duration-(--duration-base) ease-standard",
          isOn ? "translate-x-4.5" : "translate-x-0.5",
        )}
      />
    </button>
  );

  if (!label) return <span className={className}>{control}</span>;
  return (
    <div className={cn("flex items-start justify-between gap-4", className)}>
      <span className="flex flex-col gap-0.5">
        <label htmlFor={controlId} className="text-sm font-medium text-fg select-none">
          {label}
        </label>
        {description ? (
          <span id={`${controlId}-description`} className="text-sm text-fg-muted">
            {description}
          </span>
        ) : null}
      </span>
      {control}
    </div>
  );
}
