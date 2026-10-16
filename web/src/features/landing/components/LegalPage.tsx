import Link from "next/link";

import { AlertTriangleIcon } from "@/components/ui";

import { PUBLIC_ROUTES } from "../content";
import { LEGAL_CONFIG, type LegalDocument } from "../legal";
import { PageHeader } from "./PageHeader";

function formatDate(iso: string): string {
  return new Date(`${iso}T00:00:00Z`).toLocaleDateString("en-US", {
    year: "numeric",
    month: "long",
    day: "numeric",
    timeZone: "UTC",
  });
}

/** Privacy Policy / Terms of Service layout: table of contents, numbered sections. */
export function LegalPage({ document }: { document: LegalDocument }) {
  return (
    <>
      <PageHeader eyebrow="Legal" title={document.title} description={document.summary}>
        {LEGAL_CONFIG.lastUpdated ? (
          <p className="mt-3 text-sm text-fg-subtle">
            Last updated {formatDate(LEGAL_CONFIG.lastUpdated)}
          </p>
        ) : null}
      </PageHeader>
      <div className="mx-auto flex max-w-3xl flex-col gap-10 px-4 py-12 sm:px-6">
        {LEGAL_CONFIG.reviewed ? null : (
          <div
            role="note"
            data-testid="legal-draft-notice"
            className="flex gap-3 rounded-lg bg-warning-soft p-4 text-sm text-warning-text"
          >
            <AlertTriangleIcon size={18} className="mt-0.5 shrink-0" />
            <p>
              Draft: this document is a template and has not yet been reviewed by a lawyer. It may
              change before the platform launches.
            </p>
          </div>
        )}
        <nav aria-label="Contents" className="rounded-lg border border-border bg-surface p-5">
          <h2 className="text-sm font-semibold text-fg">Contents</h2>
          <ol className="mt-3 flex list-decimal flex-col gap-1.5 pl-5 text-sm text-fg-muted">
            {document.sections.map((section) => (
              <li key={section.id}>
                <a href={`#${section.id}`} className="focus-ring rounded-sm hover:text-fg">
                  {section.title}
                </a>
              </li>
            ))}
          </ol>
        </nav>
        <div className="flex flex-col gap-8">
          {document.sections.map((section, index) => (
            <section key={section.id} id={section.id} className="scroll-mt-20">
              <h2 className="text-xl font-semibold text-fg">
                {index + 1}. {section.title}
              </h2>
              {section.paragraphs?.map((paragraph) => (
                <p key={paragraph} className="mt-3 text-base text-fg-muted">
                  {paragraph}
                </p>
              ))}
              {section.list ? (
                <ul className="mt-3 flex list-disc flex-col gap-2 pl-5 text-base text-fg-muted">
                  {section.list.map((item) => (
                    <li key={item}>{item}</li>
                  ))}
                </ul>
              ) : null}
            </section>
          ))}
        </div>
        <p className="border-t border-border pt-6 text-sm text-fg-muted">
          Questions about this document?{" "}
          <Link
            href={PUBLIC_ROUTES.contact}
            className="focus-ring rounded-sm font-medium text-primary-text underline underline-offset-4"
          >
            Contact us
          </Link>
          .
        </p>
      </div>
    </>
  );
}
