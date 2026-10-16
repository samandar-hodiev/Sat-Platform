import { ContactCard } from "@/features/landing/components/ContactCard";
import { PageHeader } from "@/features/landing/components/PageHeader";
import { PUBLIC_ROUTES } from "@/features/landing/content";
import { publicPageMetadata } from "@/features/landing/seo";
import { supportEmail } from "@/lib/site";

export const metadata = publicPageMetadata({
  path: PUBLIC_ROUTES.contact,
  title: "Contact",
  description: "How to reach the SAT Platform team with questions, feedback or account requests.",
});

export default function ContactPage() {
  return (
    <>
      <PageHeader
        eyebrow="Contact"
        title="Get in touch"
        description="Questions about the platform, your account or your data? We are happy to help."
      />
      <div className="mx-auto max-w-2xl px-4 py-14 sm:px-6">
        <ContactCard email={supportEmail()} />
      </div>
    </>
  );
}
