import Link from "next/link";

import { JsonLd } from "@/features/landing/components/JsonLd";
import { PageHeader } from "@/features/landing/components/PageHeader";
import { FaqList, PricingPlans } from "@/features/landing/components/Sections";
import { FAQ, PUBLIC_ROUTES } from "@/features/landing/content";
import { faqJsonLd, publicPageMetadata } from "@/features/landing/seo";

export const metadata = publicPageMetadata({
  path: PUBLIC_ROUTES.pricing,
  title: "Pricing",
  description:
    "Free and Premium plans for Digital SAT self-study. Start free; Premium prices will be published before launch.",
});

const PRICING_FAQ = FAQ.filter((item) => ["price", "guarantee", "scores"].includes(item.id));

export default function PricingPage() {
  return (
    <>
      <JsonLd data={faqJsonLd(PRICING_FAQ)} />
      <PageHeader
        eyebrow="Pricing"
        title="Start free, upgrade when you are ready"
        description="Free to get started and Premium for full preparation."
      />
      <div className="mx-auto max-w-6xl px-4 py-14 sm:px-6">
        <PricingPlans headingLevel={2} />
      </div>
      <section aria-labelledby="pricing-faq" className="border-t border-border bg-surface">
        <div className="mx-auto max-w-6xl px-4 py-14 sm:px-6">
          <h2
            id="pricing-faq"
            className="mb-8 text-center text-2xl font-semibold tracking-tight text-fg"
          >
            Questions about plans
          </h2>
          <FaqList items={PRICING_FAQ} />
          <p className="mt-6 text-center text-sm text-fg-muted">
            Something else?{" "}
            <Link
              href={PUBLIC_ROUTES.contact}
              className="focus-ring rounded-sm font-medium text-primary-text underline underline-offset-4"
            >
              Contact us
            </Link>
          </p>
        </div>
      </section>
    </>
  );
}
