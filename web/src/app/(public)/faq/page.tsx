import Link from "next/link";

import { JsonLd } from "@/features/landing/components/JsonLd";
import { PageHeader } from "@/features/landing/components/PageHeader";
import { FaqList } from "@/features/landing/components/Sections";
import { FAQ, PUBLIC_ROUTES } from "@/features/landing/content";
import { faqJsonLd, publicPageMetadata } from "@/features/landing/seo";

export const metadata = publicPageMetadata({
  path: PUBLIC_ROUTES.faq,
  title: "FAQ",
  description:
    "Answers about SAT Platform: mock exams, scores, the placement test, the AI study assistant, pricing and your data.",
});

export default function FaqPage() {
  return (
    <>
      <JsonLd data={faqJsonLd(FAQ)} />
      <PageHeader
        eyebrow="FAQ"
        title="Frequently asked questions"
        description="What the platform does, what it does not do, and how it works."
      />
      <div className="mx-auto max-w-6xl px-4 py-14 sm:px-6">
        <FaqList />
        <p className="mt-8 text-center text-sm text-fg-muted">
          Did not find your answer?{" "}
          <Link
            href={PUBLIC_ROUTES.contact}
            className="focus-ring rounded-sm font-medium text-primary-text underline underline-offset-4"
          >
            Contact us
          </Link>
        </p>
      </div>
    </>
  );
}
