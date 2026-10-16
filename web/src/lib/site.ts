// Public site identity and URLs (session 0.03). Used for SEO metadata, the sitemap, robots.txt
// and the Contact page. Only NEXT_PUBLIC_* values: they are compiled into the client bundle.

export const SITE_NAME = "SAT Platform";

export const SITE_DESCRIPTION =
  "Self-study preparation for the Digital SAT: a placement test, a personal study plan, " +
  "daily practice, mistake review, full-length timed mock exams and an AI study assistant.";

const DEV_SITE_URL = "http://localhost:3000";

/**
 * Canonical origin of the public site, e.g. https://example.com (NEXT_PUBLIC_SITE_URL, no
 * trailing slash). Falls back to the local dev server so development and tests need no setup;
 * production deployments must set it (5.05), otherwise canonical URLs point at localhost.
 */
export function siteUrl(): string {
  // Next.js inlines NEXT_PUBLIC_* only when accessed literally like this.
  const value = process.env.NEXT_PUBLIC_SITE_URL?.trim();
  return (value || DEV_SITE_URL).replace(/\/+$/, "");
}

/** Absolute URL for a site path ("/pricing" → "https://example.com/pricing"). */
export function absoluteUrl(path: string): string {
  return path === "/" ? siteUrl() : `${siteUrl()}${path.startsWith("/") ? path : `/${path}`}`;
}

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

/** Public support address (NEXT_PUBLIC_SUPPORT_EMAIL), or null while it is not configured. */
export function supportEmail(): string | null {
  const value = process.env.NEXT_PUBLIC_SUPPORT_EMAIL?.trim();
  return value && EMAIL_PATTERN.test(value) ? value : null;
}
