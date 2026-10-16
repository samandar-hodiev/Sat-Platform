// SEO helpers for public pages (session 0.03): per-page metadata and JSON-LD structured data.

import type { Metadata } from "next";

import { SITE_DESCRIPTION, SITE_NAME, absoluteUrl } from "@/lib/site";

import type { FaqItem } from "./content";

/** Metadata for a public page: title, description, canonical URL and Open Graph/Twitter cards. */
export function publicPageMetadata({
  path,
  title,
  description = SITE_DESCRIPTION,
}: {
  path: string;
  /** Page title without the site name; omitted on the home page. */
  title?: string;
  description?: string;
}): Metadata {
  const fullTitle = title ? `${title} · ${SITE_NAME}` : `${SITE_NAME} — Digital SAT self-study`;
  return {
    // The home page sets an absolute title; other pages use the root "%s · SAT Platform" template.
    title: title ?? { absolute: fullTitle },
    description,
    alternates: { canonical: path },
    openGraph: {
      type: "website",
      siteName: SITE_NAME,
      title: fullTitle,
      description,
      url: absoluteUrl(path),
      locale: "en_US",
    },
    twitter: { card: "summary", title: fullTitle, description },
  };
}

export function organizationJsonLd() {
  return {
    "@context": "https://schema.org",
    "@type": "Organization",
    name: SITE_NAME,
    url: absoluteUrl("/"),
    description: SITE_DESCRIPTION,
  };
}

export function websiteJsonLd() {
  return {
    "@context": "https://schema.org",
    "@type": "WebSite",
    name: SITE_NAME,
    url: absoluteUrl("/"),
  };
}

export function faqJsonLd(items: FaqItem[]) {
  return {
    "@context": "https://schema.org",
    "@type": "FAQPage",
    mainEntity: items.map((item) => ({
      "@type": "Question",
      name: item.question,
      acceptedAnswer: { "@type": "Answer", text: item.answer },
    })),
  };
}

/**
 * Serializes JSON-LD for a <script type="application/ld+json">. "<" is escaped so a string can
 * never close the script tag (Next.js JSON-LD guide).
 */
export function serializeJsonLd(data: unknown): string {
  return JSON.stringify(data).replace(/</g, "\\u003c");
}
