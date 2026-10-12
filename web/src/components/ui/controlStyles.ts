import { cn } from "@/lib/cn";

/** Shared look of text-like controls (Input, Textarea, Select). */
export function controlClasses(invalid: boolean, className?: string): string {
  return cn(
    "w-full rounded-md border bg-surface text-sm text-fg shadow-xs placeholder:text-fg-subtle",
    "transition-[border-color,box-shadow] duration-(--duration-fast) ease-standard",
    "focus-visible:outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/25",
    "disabled:cursor-not-allowed disabled:bg-surface-muted disabled:text-fg-subtle",
    "read-only:bg-surface-muted",
    invalid
      ? "border-error focus-visible:border-error focus-visible:ring-error/25"
      : "border-border-strong hover:border-fg-subtle",
    className,
  );
}
