import type { ComponentProps } from "react";

import { cn } from "@/lib/cn";

export type BadgeVariant = "neutral" | "primary" | "success" | "warning" | "error" | "info";

const variants: Record<BadgeVariant, string> = {
  neutral: "bg-surface-muted text-fg-muted border-border",
  primary: "bg-primary-soft text-primary-text border-transparent",
  success: "bg-success-soft text-success-text border-transparent",
  warning: "bg-warning-soft text-warning-text border-transparent",
  error: "bg-error-soft text-error-text border-transparent",
  info: "bg-info-soft text-info-text border-transparent",
};

export interface BadgeProps extends ComponentProps<"span"> {
  variant?: BadgeVariant;
}

export function Badge({ variant = "neutral", className, ...props }: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium whitespace-nowrap",
        variants[variant],
        className,
      )}
      {...props}
    />
  );
}
