"use client";

import type { ComponentProps } from "react";

import { controlClasses } from "./controlStyles";
import { useFieldControl } from "./Field";

export type InputProps = ComponentProps<"input">;

export function Input({ className, ...props }: InputProps) {
  const { invalid, ...field } = useFieldControl(props);
  return (
    <input
      {...props}
      {...field}
      className={controlClasses(invalid, `h-(--control-md) px-3 ${className ?? ""}`)}
    />
  );
}
