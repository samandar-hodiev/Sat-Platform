import { LegalPage } from "@/features/landing/components/LegalPage";
import { PUBLIC_ROUTES } from "@/features/landing/content";
import { TERMS_OF_SERVICE } from "@/features/landing/legal";
import { publicPageMetadata } from "@/features/landing/seo";

export const metadata = publicPageMetadata({
  path: PUBLIC_ROUTES.terms,
  title: TERMS_OF_SERVICE.title,
  description: TERMS_OF_SERVICE.summary,
});

export default function TermsPage() {
  return <LegalPage document={TERMS_OF_SERVICE} />;
}
