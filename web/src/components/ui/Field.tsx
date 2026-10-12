"use client";

import { createContext, useContext, useId, type ComponentProps, type ReactNode } from "react";

import { cn } from "@/lib/cn";

interface FieldContextValue {
  id: string;
  describedBy: string | undefined;
  invalid: boolean;
  required: boolean;
}

const FieldContext = createContext<FieldContextValue | null>(null);

/**
 * Wires a control to its label, hint and error: Input/Textarea/Select inside a FormField pick up
 * id, aria-describedby, aria-invalid and required automatically.
 */
export function useFieldControl(props: {
  id?: string;
  "aria-describedby"?: string;
  "aria-invalid"?: ComponentProps<"input">["aria-invalid"];
  required?: boolean;
}) {
  const field = useContext(FieldContext);
  const describedBy =
    [props["aria-describedby"], field?.describedBy].filter(Boolean).join(" ") || undefined;
  const invalid = props["aria-invalid"] ?? (field?.invalid ? true : undefined);
  return {
    id: props.id ?? field?.id,
    "aria-describedby": describedBy,
    "aria-invalid": invalid,
    required: props.required ?? field?.required,
    invalid: invalid === true || invalid === "true",
  };
}

export function Label({ className, ...props }: ComponentProps<"label">) {
  return <label className={cn("text-sm font-medium text-fg select-none", className)} {...props} />;
}

export interface FormFieldProps {
  label: ReactNode;
  /** Short help text under the control. */
  hint?: ReactNode;
  /** Validation message. Replaces the hint and marks the control invalid. */
  error?: ReactNode;
  required?: boolean;
  id?: string;
  className?: string;
  children: ReactNode;
}

export function FormField({
  label,
  hint,
  error,
  required = false,
  id,
  className,
  children,
}: FormFieldProps) {
  const generated = useId();
  const controlId = id ?? generated;
  const hintId = `${controlId}-hint`;
  const errorId = `${controlId}-error`;
  const describedBy = error ? errorId : hint ? hintId : undefined;

  return (
    <FieldContext.Provider
      value={{ id: controlId, describedBy, invalid: Boolean(error), required }}
    >
      <div className={cn("flex flex-col gap-1.5", className)}>
        <Label htmlFor={controlId}>
          {label}
          {required ? (
            <span className="ml-0.5 text-error-text" aria-hidden="true">
              *
            </span>
          ) : null}
        </Label>
        {children}
        {error ? (
          <p id={errorId} className="text-sm text-error-text">
            {error}
          </p>
        ) : hint ? (
          <p id={hintId} className="text-sm text-fg-muted">
            {hint}
          </p>
        ) : null}
      </div>
    </FieldContext.Provider>
  );
}
