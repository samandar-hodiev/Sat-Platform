import Link from "next/link";

import { SITE_NAME } from "@/lib/site";

import { FOOTER_GROUPS, TRADEMARK_NOTICE } from "../content";
import { PublicBrand } from "./PublicBrand";

export function PublicFooter() {
  return (
    <footer className="border-t border-border bg-surface">
      <div className="mx-auto grid max-w-6xl grid-cols-1 gap-10 px-4 py-12 sm:px-6 md:grid-cols-4">
        <div className="flex flex-col gap-3">
          <PublicBrand />
          <p className="text-sm text-fg-muted">Self-study preparation for the Digital SAT.</p>
        </div>
        {FOOTER_GROUPS.map((group) => (
          <nav key={group.title} aria-label={group.title}>
            <h2 className="text-sm font-semibold text-fg">{group.title}</h2>
            <ul className="mt-3 flex flex-col gap-2">
              {group.links.map((link) => (
                <li key={link.label}>
                  <Link
                    href={link.href}
                    className="focus-ring rounded-sm text-sm text-fg-muted hover:text-fg"
                  >
                    {link.label}
                  </Link>
                </li>
              ))}
            </ul>
          </nav>
        ))}
      </div>
      <div className="border-t border-border">
        <div className="mx-auto flex max-w-6xl flex-col gap-2 px-4 py-6 text-xs text-fg-subtle sm:px-6">
          <p>{TRADEMARK_NOTICE}</p>
          <p>
            © {new Date().getFullYear()} {SITE_NAME}
          </p>
        </div>
      </div>
    </footer>
  );
}
