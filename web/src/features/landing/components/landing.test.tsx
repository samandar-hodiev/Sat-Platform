// @vitest-environment jsdom
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import ContactPage from "@/app/(public)/contact/page";
import FaqPage from "@/app/(public)/faq/page";
import LandingPage from "@/app/(public)/page";
import PricingPage from "@/app/(public)/pricing/page";
import PrivacyPage from "@/app/(public)/privacy/page";
import TermsPage from "@/app/(public)/terms/page";
import { ThemeProvider } from "@/components/theme/ThemeProvider";
import { AUTH_ROUTES } from "@/features/auth/routes";

import { FAQ, HEADER_NAV, PLANS, SECTION_IDS } from "../content";
import { PRIVACY_POLICY } from "../legal";
import { ContactCard } from "./ContactCard";
import { PublicFooter } from "./PublicFooter";
import { PublicHeader } from "./PublicHeader";

const withTheme = (ui: React.ReactNode) => render(<ThemeProvider>{ui}</ThemeProvider>);

describe("PublicHeader", () => {
  it("links every PRD §5.1 item plus Sign In and Create Account", () => {
    withTheme(<PublicHeader />);
    const nav = screen.getByRole("navigation", { name: "Main" });
    for (const link of HEADER_NAV) {
      expect(within(nav).getByRole("link", { name: link.label }).getAttribute("href")).toBe(
        link.href,
      );
    }
    expect(screen.getByTestId("header-sign-in").getAttribute("href")).toBe(AUTH_ROUTES.signIn);
    expect(screen.getByTestId("header-sign-up").getAttribute("href")).toBe(AUTH_ROUTES.signUp);
  });

  it("opens the mobile menu and closes it when a link is chosen", async () => {
    const user = userEvent.setup();
    withTheme(<PublicHeader />);
    const button = screen.getByTestId("public-menu-button");
    const menu = screen.getByTestId("public-menu") as HTMLDialogElement;
    expect(menu.open).toBe(false);

    await user.click(button);
    expect(menu.open).toBe(true);
    expect(button.getAttribute("aria-expanded")).toBe("true");
    expect(within(menu).getByRole("link", { name: "Create Account" }).getAttribute("href")).toBe(
      AUTH_ROUTES.signUp,
    );

    await user.click(within(menu).getByRole("link", { name: "Pricing" }));
    expect(menu.open).toBe(false);
  });

  it("closes the mobile menu with the close button", async () => {
    const user = userEvent.setup();
    withTheme(<PublicHeader />);
    await user.click(screen.getByTestId("public-menu-button"));
    await user.click(screen.getByRole("button", { name: "Close menu" }));
    expect((screen.getByTestId("public-menu") as HTMLDialogElement).open).toBe(false);
  });
});

describe("PublicFooter", () => {
  it("links the legal pages and contact", () => {
    render(<PublicFooter />);
    expect(screen.getByRole("link", { name: "Privacy Policy" }).getAttribute("href")).toBe(
      "/privacy",
    );
    expect(screen.getByRole("link", { name: "Terms of Service" }).getAttribute("href")).toBe(
      "/terms",
    );
    expect(screen.getByRole("link", { name: "Contact" }).getAttribute("href")).toBe("/contact");
    expect(screen.getByText(/not affiliated with/)).toBeTruthy();
  });
});

describe("Landing page", () => {
  it("renders one h1 and every PRD §5.3 section in order", () => {
    const { container } = render(<LandingPage />);
    expect(screen.getAllByRole("heading", { level: 1 })).toHaveLength(1);
    const order = [
      SECTION_IDS.features,
      SECTION_IDS.practice,
      SECTION_IDS.mockExams,
      SECTION_IDS.assistant,
      SECTION_IDS.studyPlan,
      SECTION_IDS.progress,
      SECTION_IDS.howItWorks,
      SECTION_IDS.pricing,
      SECTION_IDS.faq,
    ];
    const ids = Array.from(container.querySelectorAll("section[id]")).map((section) => section.id);
    expect(ids).toEqual(order);
  });

  it("has Start Preparing and Explore Platform CTAs", () => {
    render(<LandingPage />);
    expect(screen.getByTestId("hero-start").getAttribute("href")).toBe(AUTH_ROUTES.signUp);
    expect(screen.getByTestId("hero-explore").getAttribute("href")).toBe(
      `/#${SECTION_IDS.features}`,
    );
  });

  it("labels product previews as illustrations, not real data", () => {
    render(<LandingPage />);
    const figures = screen.getAllByRole("img", { name: /^Illustration of/ });
    expect(figures.length).toBe(6);
    expect(screen.getAllByText("Illustration")).toHaveLength(6);
  });

  it("shows plans without prices", () => {
    render(<LandingPage />);
    for (const plan of PLANS) {
      expect(
        within(screen.getByTestId(`plan-${plan.id}`)).getByText("Pricing coming soon"),
      ).toBeTruthy();
    }
  });

  it("includes structured data", () => {
    const { container } = render(<LandingPage />);
    const script = container.querySelector('script[type="application/ld+json"]');
    const data = JSON.parse(script?.textContent ?? "[]") as Array<{ "@type": string }>;
    expect(data.map((entry) => entry["@type"])).toEqual(["Organization", "WebSite", "FAQPage"]);
  });
});

describe("FAQ page", () => {
  it("lists every question and expands an answer", async () => {
    const user = userEvent.setup();
    render(<FaqPage />);
    const first = FAQ[0]!;
    expect(screen.getAllByRole("button", { expanded: false })).toHaveLength(FAQ.length);
    const trigger = screen.getByRole("button", { name: first.question });
    expect(screen.getByText(first.answer).closest("[role=region]")?.hasAttribute("hidden")).toBe(
      true,
    );
    await user.click(trigger);
    expect(trigger.getAttribute("aria-expanded")).toBe("true");
    expect(screen.getByText(first.answer).closest("[role=region]")?.hasAttribute("hidden")).toBe(
      false,
    );
  });
});

describe("Pricing page", () => {
  it("has a single h1 and both plans", () => {
    render(<PricingPage />);
    expect(screen.getAllByRole("heading", { level: 1 })).toHaveLength(1);
    expect(screen.getByRole("heading", { level: 2, name: "Free" })).toBeTruthy();
    expect(screen.getByRole("heading", { level: 2, name: "Premium" })).toBeTruthy();
  });
});

describe("Contact", () => {
  it("shows a mailto link when the support email is configured", () => {
    render(<ContactCard email="help@example.com" />);
    expect(screen.getByRole("link", { name: "help@example.com" }).getAttribute("href")).toBe(
      "mailto:help@example.com",
    );
  });

  it("says the address is coming soon instead of inventing one", () => {
    render(<ContactCard email={null} />);
    expect(screen.getByTestId("contact-unavailable")).toBeTruthy();
    expect(screen.queryByRole("link", { name: /@/ })).toBeNull();
    expect(screen.getByRole("link", { name: "Read the FAQ" }).getAttribute("href")).toBe("/faq");
  });

  it("renders the page", () => {
    render(<ContactPage />);
    expect(screen.getByRole("heading", { level: 1, name: "Get in touch" })).toBeTruthy();
  });
});

describe("Legal pages", () => {
  it("render every section with a table of contents and a draft notice", () => {
    render(<PrivacyPage />);
    expect(screen.getByRole("heading", { level: 1, name: "Privacy Policy" })).toBeTruthy();
    expect(screen.getByTestId("legal-draft-notice")).toBeTruthy();
    const toc = screen.getByRole("navigation", { name: "Contents" });
    expect(within(toc).getAllByRole("link")).toHaveLength(PRIVACY_POLICY.sections.length);
  });

  it("terms say there is no score guarantee", () => {
    render(<TermsPage />);
    expect(screen.getByRole("heading", { name: /No score guarantee/ })).toBeTruthy();
  });
});
