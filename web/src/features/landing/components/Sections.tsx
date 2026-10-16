import Link from "next/link";
import type { ReactNode } from "react";

import {
  Accordion,
  AccordionItem,
  Badge,
  Card,
  CardContent,
  CheckIcon,
  buttonClasses,
} from "@/components/ui";
import { cn } from "@/lib/cn";

import {
  FAQ,
  FEATURES,
  HERO,
  HOW_IT_WORKS,
  PLANS,
  PRICING_STATUS,
  PUBLIC_ROUTES,
  SECTION_IDS,
  SPOTLIGHTS,
  type FaqItem,
} from "../content";
import { ArrowRightIcon, FeatureIconGlyph } from "../icons";
import { DashboardIllustration, SpotlightIllustration } from "./Illustration";

/** Shared section frame: anchor id, centered heading block, consistent spacing. */
export function Section({
  id,
  eyebrow,
  title,
  description,
  muted = false,
  children,
}: {
  id?: string;
  eyebrow?: string;
  title: string;
  description?: string;
  muted?: boolean;
  children: ReactNode;
}) {
  const headingId = id ? `${id}-heading` : undefined;
  return (
    <section
      id={id}
      aria-labelledby={headingId}
      className={cn("scroll-mt-16 border-b border-border", muted ? "bg-surface" : "bg-canvas")}
    >
      <div className="mx-auto max-w-6xl px-4 py-16 sm:px-6 sm:py-20">
        <div className="mx-auto max-w-2xl text-center">
          {eyebrow ? <p className="text-sm font-semibold text-primary-text">{eyebrow}</p> : null}
          <h2
            id={headingId}
            className="mt-2 text-3xl font-semibold tracking-tight text-fg sm:text-4xl"
          >
            {title}
          </h2>
          {description ? <p className="mt-4 text-base text-fg-muted">{description}</p> : null}
        </div>
        <div className="mt-12">{children}</div>
      </div>
    </section>
  );
}

export function Hero() {
  return (
    <section aria-labelledby="hero-heading" className="border-b border-border bg-canvas">
      <div className="mx-auto grid max-w-6xl grid-cols-1 items-center gap-12 px-4 py-16 sm:px-6 sm:py-24 lg:grid-cols-2">
        <div className="flex flex-col items-start gap-6">
          <Badge variant="primary">{HERO.eyebrow}</Badge>
          <h1
            id="hero-heading"
            className="text-4xl font-semibold tracking-tight text-balance text-fg"
          >
            {HERO.title}
          </h1>
          <p className="text-lg text-fg-muted">{HERO.description}</p>
          <div className="flex w-full flex-col gap-3 sm:w-auto sm:flex-row">
            <Link
              href={HERO.primaryCta.href}
              className={buttonClasses({ size: "lg" })}
              data-testid="hero-start"
            >
              {HERO.primaryCta.label}
              <ArrowRightIcon size={18} />
            </Link>
            <Link
              href={HERO.secondaryCta.href}
              className={buttonClasses({ size: "lg", variant: "outline" })}
              data-testid="hero-explore"
            >
              {HERO.secondaryCta.label}
            </Link>
          </div>
          <p className="text-sm text-fg-subtle">{HERO.note}</p>
        </div>
        <DashboardIllustration />
      </div>
    </section>
  );
}

export function FeatureGrid() {
  return (
    <Section
      id={SECTION_IDS.features}
      eyebrow="Features"
      title="Everything you need to prepare, in one place"
      description="Each part of the platform feeds the next: your results shape your plan, and your mistakes shape your practice."
      muted
    >
      <ul className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {FEATURES.map((feature) => (
          <li key={feature.title}>
            <Card className="h-full">
              <CardContent className="flex flex-col gap-3">
                <span className="flex size-10 items-center justify-center rounded-lg bg-primary-soft text-primary-text">
                  <FeatureIconGlyph name={feature.icon} />
                </span>
                <h3 className="text-base font-semibold text-fg">{feature.title}</h3>
                <p className="text-sm text-fg-muted">{feature.description}</p>
              </CardContent>
            </Card>
          </li>
        ))}
      </ul>
    </Section>
  );
}

/** Daily Practice, Mock Exam, AI Assistant, Study Plan, Progress (PRD §5.3, items 3–7). */
export function Spotlights() {
  return (
    <>
      {SPOTLIGHTS.map((spotlight, index) => (
        <section
          key={spotlight.id}
          id={spotlight.id}
          aria-labelledby={`${spotlight.id}-heading`}
          className={cn(
            "scroll-mt-16 border-b border-border",
            index % 2 === 0 ? "bg-canvas" : "bg-surface",
          )}
        >
          <div className="mx-auto grid max-w-6xl grid-cols-1 items-center gap-10 px-4 py-16 sm:px-6 sm:py-20 lg:grid-cols-2 lg:gap-16">
            <div className={cn("flex flex-col gap-4", index % 2 === 1 && "lg:order-2")}>
              <p className="text-sm font-semibold text-primary-text">{spotlight.eyebrow}</p>
              <h2
                id={`${spotlight.id}-heading`}
                className="text-3xl font-semibold tracking-tight text-fg"
              >
                {spotlight.title}
              </h2>
              <p className="text-base text-fg-muted">{spotlight.description}</p>
              <ul className="mt-2 flex flex-col gap-3">
                {spotlight.points.map((point) => (
                  <li key={point} className="flex gap-3 text-sm text-fg">
                    <span className="mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full bg-success-soft text-success-text">
                      <CheckIcon size={12} />
                    </span>
                    {point}
                  </li>
                ))}
              </ul>
            </div>
            <SpotlightIllustration visual={spotlight.visual} />
          </div>
        </section>
      ))}
    </>
  );
}

export function HowItWorks() {
  return (
    <Section
      id={SECTION_IDS.howItWorks}
      eyebrow="How It Works"
      title="From first sign-up to exam day"
      description="Five steps, the same loop every learner follows."
      muted
    >
      <ol className="grid grid-cols-1 gap-4 md:grid-cols-5">
        {HOW_IT_WORKS.map((step, index) => (
          <li
            key={step.title}
            className="flex flex-col gap-3 rounded-lg border border-border bg-canvas p-5"
          >
            <span className="flex size-8 items-center justify-center rounded-full bg-primary text-sm font-semibold text-primary-fg">
              {index + 1}
            </span>
            <h3 className="text-base font-semibold text-fg">{step.title}</h3>
            <p className="text-sm text-fg-muted">{step.description}</p>
          </li>
        ))}
      </ol>
    </Section>
  );
}

/** Plan cards without prices: prices and limits come from Subscription & Plans (3.13). */
export function PricingPlans({ headingLevel = 3 }: { headingLevel?: 2 | 3 }) {
  const Heading = `h${headingLevel}` as const;
  return (
    <div className="flex flex-col gap-6">
      <ul className="mx-auto grid w-full grid-cols-1 max-w-4xl gap-4 md:grid-cols-2">
        {PLANS.map((plan) => (
          <li key={plan.id} data-testid={`plan-${plan.id}`}>
            <Card
              className={cn("flex h-full flex-col", plan.featured && "border-primary shadow-md")}
            >
              <CardContent className="flex flex-1 flex-col gap-5">
                <div className="flex items-center justify-between gap-3">
                  <Heading className="text-xl font-semibold text-fg">{plan.name}</Heading>
                  {plan.featured ? <Badge variant="primary">For full preparation</Badge> : null}
                </div>
                <p className="text-sm text-fg-muted">{plan.summary}</p>
                <p className="text-lg font-semibold text-fg">{PRICING_STATUS.price}</p>
                <ul className="flex flex-1 flex-col gap-3">
                  {plan.highlights.map((item) => (
                    <li key={item} className="flex gap-3 text-sm text-fg">
                      <CheckIcon size={16} className="mt-0.5 shrink-0 text-success-text" />
                      {item}
                    </li>
                  ))}
                </ul>
                <Link
                  href={plan.cta.href}
                  className={buttonClasses({
                    fullWidth: true,
                    variant: plan.featured ? "primary" : "outline",
                  })}
                >
                  {plan.cta.label}
                </Link>
              </CardContent>
            </Card>
          </li>
        ))}
      </ul>
      <p className="mx-auto max-w-2xl text-center text-sm text-fg-subtle">{PRICING_STATUS.note}</p>
    </div>
  );
}

export function PricingSection() {
  return (
    <Section
      id={SECTION_IDS.pricing}
      eyebrow="Pricing"
      title="Start free, upgrade when you are ready"
      description="Two plans: Free to get started and Premium for full preparation."
    >
      <PricingPlans />
      <p className="mt-6 text-center">
        <Link
          href={PUBLIC_ROUTES.pricing}
          className="focus-ring rounded-sm text-sm font-medium text-primary-text underline underline-offset-4"
        >
          Compare plans
        </Link>
      </p>
    </Section>
  );
}

export function FaqList({ items = FAQ }: { items?: FaqItem[] }) {
  return (
    <Accordion type="multiple" className="mx-auto max-w-3xl">
      {items.map((item) => (
        <AccordionItem key={item.id} value={item.id} title={item.question}>
          <p>{item.answer}</p>
        </AccordionItem>
      ))}
    </Accordion>
  );
}

/** The most-asked questions on the landing page; the full list is on /faq. */
export function FaqSection() {
  return (
    <Section id={SECTION_IDS.faq} eyebrow="FAQ" title="Frequently asked questions" muted>
      <FaqList items={FAQ.slice(0, 6)} />
      <p className="mt-6 text-center">
        <Link
          href={PUBLIC_ROUTES.faq}
          className="focus-ring rounded-sm text-sm font-medium text-primary-text underline underline-offset-4"
        >
          See all questions
        </Link>
      </p>
    </Section>
  );
}

export function CtaBanner() {
  return (
    <section aria-labelledby="cta-heading" className="bg-canvas">
      <div className="mx-auto max-w-6xl px-4 py-16 sm:px-6">
        <div className="flex flex-col items-center gap-5 rounded-xl bg-primary px-6 py-12 text-center text-primary-fg">
          <h2 id="cta-heading" className="text-3xl font-semibold tracking-tight">
            Ready to start preparing?
          </h2>
          <p className="max-w-xl text-base opacity-90">
            Create a free account, take the placement test and get your first study plan.
          </p>
          <Link
            href={PUBLIC_ROUTES.signUp}
            className={buttonClasses({ size: "lg", variant: "secondary" })}
          >
            Create Account
          </Link>
        </div>
      </div>
    </section>
  );
}
