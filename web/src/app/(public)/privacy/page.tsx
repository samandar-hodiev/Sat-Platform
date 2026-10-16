import { LegalPage } from "@/features/landing/components/LegalPage";
import { PUBLIC_ROUTES } from "@/features/landing/content";
import { PRIVACY_POLICY } from "@/features/landing/legal";
import { publicPageMetadata } from "@/features/landing/seo";

export const metadata = publicPageMetadata({
  path: PUBLIC_ROUTES.privacy,
  title: PRIVACY_POLICY.title,
  description: PRIVACY_POLICY.summary,
});

export default function PrivacyPage() {
  return <LegalPage document={PRIVACY_POLICY} />;
}
