import { afterEach, describe, expect, it, vi } from "vitest";

import robots, { DISALLOWED_PATHS } from "@/app/robots";
import sitemap from "@/app/sitemap";
import { absoluteUrl, siteUrl, supportEmail } from "@/lib/site";

import { FAQ } from "./content";
import { faqJsonLd, publicPageMetadata, serializeJsonLd } from "./seo";

afterEach(() => vi.unstubAllEnvs());

describe("site config", () => {
  it("falls back to the local dev server and trims trailing slashes", () => {
    vi.stubEnv("NEXT_PUBLIC_SITE_URL", "");
    expect(siteUrl()).toBe("http://localhost:3000");
    vi.stubEnv("NEXT_PUBLIC_SITE_URL", "https://sat.example.com/");
    expect(siteUrl()).toBe("https://sat.example.com");
    expect(absoluteUrl("/")).toBe("https://sat.example.com");
    expect(absoluteUrl("/faq")).toBe("https://sat.example.com/faq");
  });

  it("returns the support email only when it is a valid address", () => {
    vi.stubEnv("NEXT_PUBLIC_SUPPORT_EMAIL", "");
    expect(supportEmail()).toBeNull();
    vi.stubEnv("NEXT_PUBLIC_SUPPORT_EMAIL", "not-an-email");
    expect(supportEmail()).toBeNull();
    vi.stubEnv("NEXT_PUBLIC_SUPPORT_EMAIL", " help@example.com ");
    expect(supportEmail()).toBe("help@example.com");
  });
});

describe("sitemap and robots", () => {
  it("lists every public page with absolute URLs and nothing private", () => {
    vi.stubEnv("NEXT_PUBLIC_SITE_URL", "https://sat.example.com");
    const urls = sitemap().map((entry) => entry.url);
    expect(urls).toEqual([
      "https://sat.example.com",
      "https://sat.example.com/pricing",
      "https://sat.example.com/faq",
      "https://sat.example.com/contact",
      "https://sat.example.com/privacy",
      "https://sat.example.com/terms",
    ]);
    for (const url of urls) {
      for (const path of DISALLOWED_PATHS) expect(url).not.toContain(`.com${path}`);
    }
  });

  it("blocks private areas and points at the sitemap", () => {
    vi.stubEnv("NEXT_PUBLIC_SITE_URL", "https://sat.example.com");
    const result = robots();
    expect(result.rules).toMatchObject({ userAgent: "*", allow: "/" });
    expect(DISALLOWED_PATHS).toEqual(expect.arrayContaining(["/app", "/owner", "/exam", "/api"]));
    expect(result.sitemap).toBe("https://sat.example.com/sitemap.xml");
  });
});

describe("page metadata", () => {
  it("sets canonical and Open Graph URLs", () => {
    vi.stubEnv("NEXT_PUBLIC_SITE_URL", "https://sat.example.com");
    const metadata = publicPageMetadata({ path: "/pricing", title: "Pricing" });
    expect(metadata.title).toBe("Pricing");
    expect(metadata.alternates?.canonical).toBe("/pricing");
    expect(metadata.openGraph).toMatchObject({
      url: "https://sat.example.com/pricing",
      title: "Pricing · SAT Platform",
    });
  });

  it("uses an absolute title on the home page", () => {
    expect(publicPageMetadata({ path: "/" }).title).toEqual({
      absolute: "SAT Platform — Digital SAT self-study",
    });
  });
});

describe("JSON-LD", () => {
  it("builds a FAQPage from the FAQ", () => {
    const data = faqJsonLd(FAQ);
    expect(data["@type"]).toBe("FAQPage");
    expect(data.mainEntity).toHaveLength(FAQ.length);
    expect(data.mainEntity[0]).toMatchObject({
      "@type": "Question",
      acceptedAnswer: { "@type": "Answer" },
    });
  });

  it("escapes < so a value cannot close the script tag", () => {
    const json = serializeJsonLd({ name: "</script><script>alert(1)</script>" });
    expect(json).not.toContain("<");
    expect(JSON.parse(json)).toEqual({ name: "</script><script>alert(1)</script>" });
  });
});
