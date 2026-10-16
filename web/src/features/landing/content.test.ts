import { describe, expect, it } from "vitest";

import { AUTH_ROUTES } from "@/features/auth/routes";

import {
  FAQ,
  FEATURES,
  FOOTER_GROUPS,
  HEADER_NAV,
  HERO,
  HOW_IT_WORKS,
  PLANS,
  PRICING_STATUS,
  PUBLIC_ROUTES,
  SECTION_IDS,
  SPOTLIGHTS,
  TRADEMARK_NOTICE,
} from "./content";
import { PRIVACY_POLICY, TERMS_OF_SERVICE } from "./legal";

function allCopy(): string {
  return JSON.stringify({
    HERO,
    FEATURES,
    SPOTLIGHTS,
    HOW_IT_WORKS,
    PLANS,
    PRICING_STATUS,
    FAQ,
    TRADEMARK_NOTICE,
  });
}

describe("landing copy rules (PRD §5.2, CLAUDE.md)", () => {
  it("makes no score guarantees or promised point gains", () => {
    const copy = allCopy().toLowerCase();
    expect(copy).not.toMatch(/guaranteed|we guarantee|boost your score|\+\s?\d+\s?points/);
  });

  it("mentions official scores only to say the platform does not give them", () => {
    const sentences = allCopy()
      .toLowerCase()
      .match(/[^.?!"]+[.?!]?/g)!
      // Questions (FAQ) may ask about it; statements must deny it.
      .filter((sentence) => /official (sat )?score/.test(sentence) && !sentence.endsWith("?"));
    expect(sentences.length).toBeGreaterThan(0);
    for (const sentence of sentences) expect(sentence).toMatch(/\b(not|no|only)\b/);
  });

  it("contains no invented prices, counts or ratings", () => {
    const copy = allCopy();
    expect(copy).not.toMatch(/[$€£]\s?\d|\d+\s?(usd|uzs|so'm)/i);
    expect(copy).not.toMatch(/\d[\d,]*\+?\s?(learners|students|users|reviews)/i);
    expect(copy).not.toMatch(/\d(\.\d)?\s?\/\s?5|★/);
  });

  it("states that scores are estimates, not official", () => {
    expect(HERO.note).toMatch(/not official SAT scores/);
    expect(TRADEMARK_NOTICE).toMatch(/not affiliated/);
    expect(FAQ.find((item) => item.id === "scores")?.answer).toMatch(/estimated score range/);
  });
});

describe("landing structure", () => {
  it("header follows PRD §5.1", () => {
    expect(HEADER_NAV.map((link) => link.label)).toEqual([
      "How It Works",
      "Features",
      "Practice",
      "Mock Exams",
      "Pricing",
      "FAQ",
    ]);
  });

  it("anchors in links point at existing section ids", () => {
    const ids = new Set<string>(Object.values(SECTION_IDS));
    const links = [...HEADER_NAV, ...FOOTER_GROUPS.flatMap((group) => group.links)];
    for (const link of links) {
      const anchor = link.href.split("#")[1];
      if (anchor) expect(ids.has(anchor), link.href).toBe(true);
    }
    for (const spotlight of SPOTLIGHTS) expect(ids.has(spotlight.id)).toBe(true);
  });

  it("CTAs go to the learner auth pages", () => {
    expect(HERO.primaryCta.href).toBe(AUTH_ROUTES.signUp);
    expect(PUBLIC_ROUTES.signIn).toBe(AUTH_ROUTES.signIn);
    expect(PLANS.every((plan) => plan.cta.href === AUTH_ROUTES.signUp)).toBe(true);
  });

  it("legal routes match what the auth screens link to", () => {
    expect(PUBLIC_ROUTES.terms).toBe(AUTH_ROUTES.terms);
    expect(PUBLIC_ROUTES.privacy).toBe(AUTH_ROUTES.privacy);
  });

  it("footer links Privacy Policy, Terms of Service and Contact (PRD §5.3)", () => {
    const hrefs = FOOTER_GROUPS.flatMap((group) => group.links.map((link) => link.href));
    expect(hrefs).toEqual(
      expect.arrayContaining([PUBLIC_ROUTES.privacy, PUBLIC_ROUTES.terms, PUBLIC_ROUTES.contact]),
    );
  });

  it("FAQ ids and legal section ids are unique", () => {
    const unique = (values: string[]) => new Set(values).size === values.length;
    expect(unique(FAQ.map((item) => item.id))).toBe(true);
    expect(unique(PRIVACY_POLICY.sections.map((section) => section.id))).toBe(true);
    expect(unique(TERMS_OF_SERVICE.sections.map((section) => section.id))).toBe(true);
  });
});
