import { JsonLd } from "@/features/landing/components/JsonLd";
import {
  CtaBanner,
  FaqSection,
  FeatureGrid,
  Hero,
  HowItWorks,
  PricingSection,
  Spotlights,
} from "@/features/landing/components/Sections";
import { FAQ } from "@/features/landing/content";
import {
  faqJsonLd,
  organizationJsonLd,
  publicPageMetadata,
  websiteJsonLd,
} from "@/features/landing/seo";

export const metadata = publicPageMetadata({ path: "/" });

// Landing page, PRD §5.3 section order.
export default function LandingPage() {
  return (
    <>
      <JsonLd data={[organizationJsonLd(), websiteJsonLd(), faqJsonLd(FAQ.slice(0, 6))]} />
      <Hero />
      <FeatureGrid />
      <Spotlights />
      <HowItWorks />
      <PricingSection />
      <FaqSection />
      <CtaBanner />
    </>
  );
}
