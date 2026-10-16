import type { MetadataRoute } from "next";

import { PUBLIC_ROUTES } from "@/features/landing/content";
import { absoluteUrl } from "@/lib/site";

// Public, indexable pages only (session 0.03). Learner, Owner, auth and exam pages need an
// account and are excluded here and in robots.ts.
const PAGES: Array<{
  path: string;
  changeFrequency: "weekly" | "monthly" | "yearly";
  priority: number;
}> = [
  { path: PUBLIC_ROUTES.home, changeFrequency: "weekly", priority: 1 },
  { path: PUBLIC_ROUTES.pricing, changeFrequency: "monthly", priority: 0.8 },
  { path: PUBLIC_ROUTES.faq, changeFrequency: "monthly", priority: 0.7 },
  { path: PUBLIC_ROUTES.contact, changeFrequency: "yearly", priority: 0.4 },
  { path: PUBLIC_ROUTES.privacy, changeFrequency: "yearly", priority: 0.3 },
  { path: PUBLIC_ROUTES.terms, changeFrequency: "yearly", priority: 0.3 },
];

export default function sitemap(): MetadataRoute.Sitemap {
  return PAGES.map((page) => ({
    url: absoluteUrl(page.path),
    changeFrequency: page.changeFrequency,
    priority: page.priority,
  }));
}
