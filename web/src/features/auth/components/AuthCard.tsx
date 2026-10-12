import type { ReactNode } from "react";

import { Card } from "@/components/ui";

/** The card every auth screen sits in: heading, short description, form, and a footer line. */
export function AuthCard({
  icon,
  title,
  description,
  children,
  footer,
}: {
  icon?: ReactNode;
  title: ReactNode;
  description?: ReactNode;
  children: ReactNode;
  /** Shown under the card, e.g. "Don't have an account? Create account". */
  footer?: ReactNode;
}) {
  return (
    <div className="flex w-full flex-col gap-6">
      <Card className="flex flex-col gap-6 p-6 shadow-sm sm:p-8">
        <div className="flex flex-col items-center gap-2 text-center">
          {icon ? (
            <span className="mb-2 flex size-11 items-center justify-center rounded-full bg-surface-muted text-fg-muted">
              {icon}
            </span>
          ) : null}
          <h1 className="text-2xl font-semibold tracking-tight text-fg">{title}</h1>
          {description ? <p className="text-sm text-fg-muted">{description}</p> : null}
        </div>
        {children}
      </Card>
      {footer ? <div className="text-center text-sm text-fg-muted">{footer}</div> : null}
    </div>
  );
}
