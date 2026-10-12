import type { ReactNode } from "react";

import { cn } from "@/lib/cn";

import { Button } from "./Button";
import { AlertCircleIcon, InboxIcon } from "./icons";
import { Spinner } from "./Spinner";

// The three non-success states every data view needs (docs/ARCHITECTURE.md → State conventions).

function StateLayout({
  icon,
  title,
  description,
  action,
  tone,
  role,
  className,
}: {
  icon: ReactNode;
  title: ReactNode;
  description?: ReactNode;
  action?: ReactNode;
  tone: "neutral" | "error";
  role?: "alert" | "status";
  className?: string;
}) {
  return (
    <div
      role={role}
      className={cn("flex flex-col items-center gap-3 px-6 py-10 text-center", className)}
    >
      <span
        className={cn(
          "flex size-10 items-center justify-center rounded-full",
          tone === "error" ? "bg-error-soft text-error-text" : "bg-surface-muted text-fg-muted",
        )}
      >
        {icon}
      </span>
      <div className="flex max-w-sm flex-col gap-1">
        <p className="text-base font-semibold text-fg">{title}</p>
        {description ? <p className="text-sm text-fg-muted">{description}</p> : null}
      </div>
      {action ? <div className="mt-1">{action}</div> : null}
    </div>
  );
}

export interface EmptyStateProps {
  title: ReactNode;
  description?: ReactNode;
  /** Usually the next useful step, e.g. "Start practice". */
  action?: ReactNode;
  icon?: ReactNode;
  className?: string;
}

export function EmptyState({ title, description, action, icon, className }: EmptyStateProps) {
  return (
    <StateLayout
      tone="neutral"
      icon={icon ?? <InboxIcon size={20} />}
      title={title}
      description={description}
      action={action}
      className={className}
    />
  );
}

export interface ErrorStateProps {
  title?: ReactNode;
  /** A user-facing explanation. Never pass raw error messages: they can leak internals. */
  description?: ReactNode;
  onRetry?: () => void;
  retrying?: boolean;
  /** Request id or error digest that helps support find the server log. */
  reference?: string;
  className?: string;
}

export function ErrorState({
  title = "Something went wrong",
  description = "Please try again. If it keeps happening, contact support.",
  onRetry,
  retrying = false,
  reference,
  className,
}: ErrorStateProps) {
  return (
    <StateLayout
      tone="error"
      role="alert"
      icon={<AlertCircleIcon size={20} />}
      title={title}
      description={
        <>
          {description}
          {reference ? (
            <span className="mt-1 block font-mono text-xs text-fg-subtle">
              Reference: {reference}
            </span>
          ) : null}
        </>
      }
      action={
        onRetry ? (
          <Button variant="outline" size="sm" onClick={onRetry} loading={retrying}>
            Try again
          </Button>
        ) : null
      }
      className={className}
    />
  );
}

export interface LoadingStateProps {
  label?: string;
  className?: string;
}

/** Centered spinner for small regions. Prefer Skeletons that match the final layout for pages. */
export function LoadingState({ label = "Loading…", className }: LoadingStateProps) {
  return (
    <div
      role="status"
      aria-live="polite"
      className={cn(
        "flex items-center justify-center gap-2 px-6 py-10 text-sm text-fg-muted",
        className,
      )}
    >
      <Spinner label={null} />
      <span>{label}</span>
    </div>
  );
}
