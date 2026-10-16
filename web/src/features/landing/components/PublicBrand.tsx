import Link from "next/link";

import { SITE_NAME } from "@/lib/site";

/** Logo mark + name linking to the landing page. */
export function PublicBrand({ onNavigate }: { onNavigate?: () => void }) {
  return (
    <Link
      href="/"
      onClick={onNavigate}
      className="focus-ring flex shrink-0 items-center gap-2 rounded-md text-base font-semibold tracking-tight text-fg"
    >
      <span className="flex size-7 items-center justify-center rounded-md bg-primary text-sm font-bold text-primary-fg">
        S
      </span>
      {SITE_NAME}
    </Link>
  );
}
