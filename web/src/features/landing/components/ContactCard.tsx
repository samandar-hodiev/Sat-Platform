import Link from "next/link";

import { Card, CardContent, EmptyState, buttonClasses } from "@/components/ui";

import { PUBLIC_ROUTES } from "../content";
import { MailIcon } from "../icons";

/**
 * Contact by email (no form, no backend). The address comes from NEXT_PUBLIC_SUPPORT_EMAIL;
 * until it is configured the page says so instead of showing a made-up address.
 */
export function ContactCard({ email }: { email: string | null }) {
  if (!email) {
    return (
      <Card data-testid="contact-unavailable">
        <EmptyState
          icon={<MailIcon />}
          title="Support email coming soon"
          description="Our support address has not been published yet. In the meantime, the FAQ answers the most common questions."
          action={
            <Link
              href={PUBLIC_ROUTES.faq}
              className={buttonClasses({ variant: "outline", size: "sm" })}
            >
              Read the FAQ
            </Link>
          }
        />
      </Card>
    );
  }

  return (
    <Card data-testid="contact-email">
      <CardContent className="flex flex-col items-center gap-4 py-10 text-center">
        <span className="flex size-12 items-center justify-center rounded-full bg-primary-soft text-primary-text">
          <MailIcon />
        </span>
        <h2 className="text-xl font-semibold text-fg">Email us</h2>
        <p className="max-w-md text-sm text-fg-muted">
          Write to us about the platform, your account, your data or a problem you found. Please do
          not send passwords or verification codes.
        </p>
        <a href={`mailto:${email}`} className={buttonClasses({ size: "lg" })}>
          {email}
        </a>
        <p className="text-sm text-fg-subtle">
          Many answers are already in the{" "}
          <Link
            href={PUBLIC_ROUTES.faq}
            className="focus-ring rounded-sm font-medium text-primary-text underline underline-offset-4"
          >
            FAQ
          </Link>
          .
        </p>
      </CardContent>
    </Card>
  );
}
