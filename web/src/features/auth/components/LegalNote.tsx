import Link from "next/link";

import { AUTH_ROUTES } from "../routes";

export function LegalNote() {
  const link = "text-fg-muted underline underline-offset-4 hover:text-fg";
  return (
    <p className="mx-auto mt-3 max-w-xs text-xs text-fg-subtle">
      By continuing, you agree to our{" "}
      <Link href={AUTH_ROUTES.terms} className={link}>
        Terms
      </Link>{" "}
      and{" "}
      <Link href={AUTH_ROUTES.privacy} className={link}>
        Privacy Policy
      </Link>
      .
    </p>
  );
}
