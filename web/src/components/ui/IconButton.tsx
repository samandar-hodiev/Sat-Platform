import type { ComponentProps } from "react";

import { cn } from "@/lib/cn";

import type { ButtonVariant, ButtonSize } from "./Button";

const variants: Record<Exclude<ButtonVariant, "danger">, string> = {
  primary: "bg-primary text-primary-fg hover:bg-primary-hover",
  secondary: "bg-secondary text-secondary-fg hover:bg-secondary-hover",
  outline: "border border-border-strong bg-surface text-fg hover:bg-surface-hover",
  ghost: "text-fg-muted hover:bg-surface-hover hover:text-fg",
};

const sizes: Record<ButtonSize, string> = {
  sm: "size-(--control-sm)",
  md: "size-(--control-md)",
  lg: "size-(--control-lg)",
};

export interface IconButtonProps extends ComponentProps<"button"> {
  /** Required: an icon-only button has no visible text for screen readers. */
  "aria-label": string;
  variant?: Exclude<ButtonVariant, "danger">;
  size?: ButtonSize;
}

export function IconButton({
  variant = "ghost",
  size = "md",
  type = "button",
  className,
  ...props
}: IconButtonProps) {
  return (
    <button
      type={type}
      className={cn(
        "focus-ring inline-flex shrink-0 items-center justify-center rounded-md",
        "transition-colors duration-(--duration-fast) ease-standard disabled:pointer-events-none disabled:opacity-50",
        variants[variant],
        sizes[size],
        className,
      )}
      {...props}
    />
  );
}
