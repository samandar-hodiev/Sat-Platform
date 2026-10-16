import type { ReactNode } from "react";

/** Title block for public sub-pages (Pricing, FAQ, Contact, legal pages). */
export function PageHeader({
  eyebrow,
  title,
  description,
  children,
}: {
  eyebrow?: string;
  title: string;
  description?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <div className="border-b border-border bg-surface">
      <div className="mx-auto max-w-3xl px-4 py-14 text-center sm:px-6 sm:py-16">
        {eyebrow ? <p className="text-sm font-semibold text-primary-text">{eyebrow}</p> : null}
        <h1 className="mt-2 text-3xl font-semibold tracking-tight text-fg sm:text-4xl">{title}</h1>
        {description ? <p className="mt-4 text-base text-fg-muted">{description}</p> : null}
        {children}
      </div>
    </div>
  );
}
