import { cn } from "@/lib/cn";

import type { OnboardingStep } from "../api";

export const STEP_ORDER: OnboardingStep[] = ["goal", "schedule", "background"];

export const STEP_TITLES: Record<OnboardingStep, string> = {
  goal: "Your goal",
  schedule: "Study time",
  background: "Where you are now",
};

/** "Step 2 of 3" with a segmented bar and the step names (PRD §8.3 progress indicator). */
export function OnboardingProgress({ current }: { current: OnboardingStep }) {
  const index = STEP_ORDER.indexOf(current);
  return (
    <nav aria-label="Onboarding progress" className="flex flex-col gap-2">
      <p className="text-sm text-fg-muted">
        Step {index + 1} of {STEP_ORDER.length}
      </p>
      <ol className="grid grid-cols-3 gap-2">
        {STEP_ORDER.map((step, i) => (
          <li
            key={step}
            aria-current={i === index ? "step" : undefined}
            className="flex flex-col gap-1.5"
          >
            <span
              aria-hidden="true"
              className={cn("h-1.5 rounded-full", i <= index ? "bg-primary" : "bg-surface-muted")}
            />
            <span className={cn("text-xs", i === index ? "font-medium text-fg" : "text-fg-muted")}>
              {STEP_TITLES[step]}
              {i < index ? <span className="sr-only"> (done)</span> : null}
            </span>
          </li>
        ))}
      </ol>
    </nav>
  );
}
