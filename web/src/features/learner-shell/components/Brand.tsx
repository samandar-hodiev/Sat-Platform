import Link from "next/link";

import { cn } from "@/lib/cn";

import { LEARNER_HOME } from "../nav";

/** Logo mark + name linking to the learner Dashboard. `compact` keeps only the mark. */
export function Brand({
  compact = false,
  onNavigate,
}: {
  compact?: boolean;
  onNavigate?: () => void;
}) {
  return (
    <Link
      href={LEARNER_HOME}
      onClick={onNavigate}
      aria-label={compact ? "SAT Platform — Dashboard" : undefined}
      className={cn(
        "focus-ring flex items-center gap-2 rounded-md text-base font-semibold tracking-tight text-fg",
        compact && "justify-center",
      )}
    >
      <span className="flex size-7 shrink-0 items-center justify-center rounded-md bg-primary text-sm font-bold text-primary-fg">
        S
      </span>
      {compact ? null : <span className="truncate">SAT Platform</span>}
    </Link>
  );
}
