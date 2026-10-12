"use client";

import { useId, type ReactNode } from "react";

import { cn } from "@/lib/cn";

// Compact option buttons for short answer sets (daily time, weekdays, 1–5 scales). Each option is
// a native radio or checkbox, visually replaced by a chip, so keyboard (arrows / space), forms and
// screen readers behave as usual.

interface ChipOption<T> {
  value: T;
  label: ReactNode;
  /** Accessible name when the visible label is short (e.g. "Mon" → "Monday"). */
  ariaLabel?: string;
}

interface GroupProps<T> {
  legend: ReactNode;
  hint?: ReactNode;
  error?: string;
  required?: boolean;
  options: ChipOption<T>[];
  /** Tailwind grid columns, e.g. "grid-cols-3 sm:grid-cols-6". */
  columns?: string;
  /** Extra content next to the legend (e.g. a Skip button). */
  action?: ReactNode;
  disabled?: boolean;
}

const chip =
  "flex min-h-(--control-md) cursor-pointer items-center justify-center rounded-md border px-3 py-2 text-center text-sm font-medium " +
  "border-border-strong bg-surface text-fg transition-colors duration-(--duration-fast) ease-standard hover:bg-surface-hover " +
  "peer-checked:border-primary peer-checked:bg-primary peer-checked:text-primary-fg peer-checked:hover:bg-primary-hover " +
  "peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-ring " +
  "peer-disabled:cursor-not-allowed peer-disabled:opacity-50";

function Frame<T>({
  legend,
  hint,
  error,
  required,
  columns,
  action,
  children,
}: Omit<GroupProps<T>, "options" | "disabled"> & { children: ReactNode }) {
  const id = useId();
  const describedBy = error ? `${id}-error` : hint ? `${id}-hint` : undefined;
  return (
    // The legend must be the fieldset's first child to name the group; the action sits in the
    // hint row, outside the legend, so it is not read as part of the question.
    <fieldset className="flex min-w-0 flex-col gap-2" aria-describedby={describedBy}>
      <legend className="text-sm font-medium text-fg">
        {legend}
        {required ? (
          <span className="ml-0.5 text-error-text" aria-hidden="true">
            *
          </span>
        ) : null}
      </legend>
      {hint || action ? (
        <div className="-mt-1 flex min-h-(--control-sm) items-center justify-between gap-2">
          {hint && !error ? (
            <p id={`${id}-hint`} className="text-sm text-fg-muted">
              {hint}
            </p>
          ) : (
            <span />
          )}
          {action}
        </div>
      ) : null}
      <div className={cn("grid gap-2", columns ?? "grid-cols-2 sm:grid-cols-3")}>{children}</div>
      {error ? (
        <p id={`${id}-error`} className="text-sm text-error-text">
          {error}
        </p>
      ) : null}
    </fieldset>
  );
}

function Chip({
  type,
  name,
  value,
  checked,
  disabled,
  ariaLabel,
  label,
  onChange,
}: {
  type: "radio" | "checkbox";
  name: string;
  value: string;
  checked: boolean;
  disabled?: boolean;
  ariaLabel?: string;
  label: ReactNode;
  onChange: () => void;
}) {
  return (
    <label className="relative block">
      <input
        type={type}
        name={name}
        value={value}
        checked={checked}
        disabled={disabled}
        aria-label={ariaLabel}
        onChange={onChange}
        className="peer sr-only"
      />
      <span className={chip}>{label}</span>
    </label>
  );
}

/** One answer. `value` null: nothing chosen yet. */
export function RadioChips<T extends string | number>({
  value,
  onChange,
  options,
  disabled,
  ...frame
}: GroupProps<T> & { value: T | null; onChange: (v: T) => void }) {
  const name = useId();
  return (
    <Frame {...frame}>
      {options.map((o) => (
        <Chip
          key={String(o.value)}
          type="radio"
          name={name}
          value={String(o.value)}
          checked={value === o.value}
          disabled={disabled}
          ariaLabel={o.ariaLabel}
          label={o.label}
          onChange={() => onChange(o.value)}
        />
      ))}
    </Frame>
  );
}

/** Several answers, kept in the order of `options`. */
export function CheckboxChips<T extends string | number>({
  values,
  onChange,
  options,
  disabled,
  ...frame
}: GroupProps<T> & { values: T[]; onChange: (v: T[]) => void }) {
  const name = useId();
  return (
    <Frame {...frame}>
      {options.map((o) => {
        const checked = values.includes(o.value);
        return (
          <Chip
            key={String(o.value)}
            type="checkbox"
            name={name}
            value={String(o.value)}
            checked={checked}
            disabled={disabled}
            ariaLabel={o.ariaLabel}
            label={o.label}
            onChange={() =>
              onChange(
                options
                  .map((x) => x.value)
                  .filter((v) => (v === o.value ? !checked : values.includes(v))),
              )
            }
          />
        );
      })}
    </Frame>
  );
}
