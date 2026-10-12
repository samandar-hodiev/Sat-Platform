"use client";

import { useId, type ComponentProps, type ReactNode } from "react";

import { cn } from "@/lib/cn";

import { CheckIcon } from "./icons";

type ChoiceProps = Omit<ComponentProps<"input">, "type"> & {
  label?: ReactNode;
  description?: ReactNode;
};

function ChoiceLayout({
  id,
  label,
  description,
  disabled,
  className,
  control,
}: {
  id: string;
  label?: ReactNode;
  description?: ReactNode;
  disabled?: boolean;
  className?: string;
  control: ReactNode;
}) {
  if (!label) return <span className={className}>{control}</span>;
  return (
    <div className={cn("flex items-start gap-2.5", disabled && "opacity-50", className)}>
      <span className="flex h-5 items-center">{control}</span>
      <span className="flex flex-col gap-0.5">
        <label htmlFor={id} className="text-sm font-medium text-fg select-none">
          {label}
        </label>
        {description ? (
          <span id={`${id}-description`} className="text-sm text-fg-muted">
            {description}
          </span>
        ) : null}
      </span>
    </div>
  );
}

const box =
  "peer size-4 shrink-0 appearance-none border border-border-strong bg-surface " +
  "transition-colors duration-(--duration-fast) checked:border-primary checked:bg-primary " +
  "disabled:cursor-not-allowed focus-ring aria-invalid:border-error";

/** Native checkbox (form submission, keyboard and indeterminate support for free). */
export function Checkbox({ id, label, description, className, disabled, ...props }: ChoiceProps) {
  const generated = useId();
  const controlId = id ?? generated;
  return (
    <ChoiceLayout
      id={controlId}
      label={label}
      description={description}
      disabled={disabled}
      className={className}
      control={
        <span className="relative inline-flex">
          <input
            id={controlId}
            type="checkbox"
            disabled={disabled}
            aria-describedby={description ? `${controlId}-description` : undefined}
            className={cn(box, "rounded-sm")}
            {...props}
          />
          <CheckIcon
            size={12}
            strokeWidth={3}
            className="pointer-events-none absolute top-1/2 left-1/2 -translate-1/2 text-primary-fg opacity-0 peer-checked:opacity-100"
          />
        </span>
      }
    />
  );
}

export function Radio({ id, label, description, className, disabled, ...props }: ChoiceProps) {
  const generated = useId();
  const controlId = id ?? generated;
  return (
    <ChoiceLayout
      id={controlId}
      label={label}
      description={description}
      disabled={disabled}
      className={className}
      control={
        <span className="relative inline-flex">
          <input
            id={controlId}
            type="radio"
            disabled={disabled}
            aria-describedby={description ? `${controlId}-description` : undefined}
            className={cn(box, "rounded-full")}
            {...props}
          />
          <span className="pointer-events-none absolute top-1/2 left-1/2 size-1.5 -translate-1/2 rounded-full bg-primary-fg opacity-0 peer-checked:opacity-100" />
        </span>
      }
    />
  );
}

/** Groups radios (or checkboxes) under one accessible name. */
export function ChoiceGroup({
  legend,
  hint,
  className,
  children,
}: {
  legend: ReactNode;
  hint?: ReactNode;
  className?: string;
  children: ReactNode;
}) {
  return (
    <fieldset className={cn("flex flex-col gap-3", className)}>
      <legend className="mb-1 text-sm font-medium text-fg">{legend}</legend>
      {hint ? <p className="-mt-2 text-sm text-fg-muted">{hint}</p> : null}
      {children}
    </fieldset>
  );
}
