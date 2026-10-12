import { cn } from "@/lib/cn";

const sizes = { sm: "size-3.5", md: "size-4", lg: "size-6" } as const;

export interface SpinnerProps {
  size?: keyof typeof sizes;
  /** Announced to screen readers. Pass null when a parent already announces the busy state. */
  label?: string | null;
  className?: string;
}

export function Spinner({ size = "md", label = "Loading", className }: SpinnerProps) {
  return (
    <span
      role={label ? "status" : undefined}
      aria-label={label ?? undefined}
      aria-hidden={label ? undefined : true}
      className={cn(
        "inline-block shrink-0 animate-spin rounded-full border-2 border-current border-r-transparent",
        sizes[size],
        className,
      )}
    />
  );
}
