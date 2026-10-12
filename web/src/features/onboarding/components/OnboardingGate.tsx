"use client";

import { useRouter } from "next/navigation";
import { useEffect, type ReactNode } from "react";

import { ErrorState, LoadingState } from "@/components/ui";

import { useOnboarding } from "../onboardingStore";
import { ONBOARDING_ROUTE } from "../routes";

/**
 * Learner app pages need finished onboarding (decided for 2.02: it is required, only optional
 * questions can be skipped). Rendered inside ProfileGate, so the profile is already complete;
 * an unfinished onboarding goes to /setup/onboarding, which resumes at the saved step.
 */
export function OnboardingGate({ children }: { children: ReactNode }) {
  const router = useRouter();
  const { state, reload } = useOnboarding();
  const incomplete = state.status === "ready" && !state.onboarding.completed;

  useEffect(() => {
    if (incomplete) router.replace(ONBOARDING_ROUTE);
  }, [incomplete, router]);

  if (state.status === "error") {
    return (
      <div className="flex justify-center p-6">
        <ErrorState
          title="We couldn't load your study goals"
          description="Check your connection and try again."
          onRetry={reload}
        />
      </div>
    );
  }
  if (state.status !== "ready" || incomplete) {
    return (
      <div className="flex justify-center p-6">
        <LoadingState label={incomplete ? "Opening onboarding…" : "Loading…"} />
      </div>
    );
  }
  return <>{children}</>;
}
