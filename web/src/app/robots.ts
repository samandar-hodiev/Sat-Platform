import type { MetadataRoute } from "next";

import { absoluteUrl } from "@/lib/site";

/** Private areas: signed-in learner app, exam runner, setup, Owner dashboard, API, dev pages. */
export const DISALLOWED_PATHS = ["/app", "/exam", "/setup", "/owner", "/api", "/dev"];

export default function robots(): MetadataRoute.Robots {
  return {
    rules: { userAgent: "*", allow: "/", disallow: DISALLOWED_PATHS },
    sitemap: absoluteUrl("/sitemap.xml"),
  };
}
