"use client";

import type { ComponentProps } from "react";

import { controlClasses } from "./controlStyles";
import { useFieldControl } from "./Field";

export type TextareaProps = ComponentProps<"textarea">;

export function Textarea({ className, rows = 4, ...props }: TextareaProps) {
  const { invalid, ...field } = useFieldControl(props);
  return (
    <textarea
      rows={rows}
      {...props}
      {...field}
      className={controlClasses(invalid, `min-h-20 px-3 py-2 ${className ?? ""}`)}
    />
  );
}
