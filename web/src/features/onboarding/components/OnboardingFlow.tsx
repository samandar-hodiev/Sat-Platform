"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

import { Card, ErrorState, LoadingState } from "@/components/ui";
import { useLearnerProfile } from "@/features/profile";
import { PROFILE_SETUP_ROUTE } from "@/features/profile/routes";
import { ApiError } from "@/lib/api/client";

import {
  completeOnboarding,
  saveBackground,
  saveGoal,
  saveSchedule,
  type BackgroundInput,
  type GoalInput,
  type Onboarding,
  type OnboardingStep,
  type ScheduleInput,
} from "../api";
import { setOnboarding, useOnboarding } from "../onboardingStore";
import { AFTER_ONBOARDING } from "../routes";
import { BackgroundStep } from "./BackgroundStep";
import { GoalStep } from "./GoalStep";
import { OnboardingProgress, STEP_ORDER } from "./OnboardingProgress";
import { ScheduleStep } from "./ScheduleStep";
import { saveFailure, type SaveFailure, type StepSave } from "./StepForm";

/**
 * The /setup/onboarding page body (PRD §8): needs a finished Profile Setup, resumes at the first
 * unsaved step, and continues to Placement (AFTER_ONBOARDING) once finished.
 */
export function OnboardingFlow() {
  const router = useRouter();
  const profile = useLearnerProfile();
  const { state, reload } = useOnboarding();
  const needsProfile = profile.state.status === "ready" && !profile.state.profile.profile_completed;
  const completed = state.status === "ready" && state.onboarding.completed;

  useEffect(() => {
    if (needsProfile) router.replace(PROFILE_SETUP_ROUTE);
    else if (completed) router.replace(AFTER_ONBOARDING);
  }, [needsProfile, completed, router]);

  if (profile.state.status === "error" || state.status === "error") {
    return (
      <ErrorState
        title="We couldn't load your answers"
        description="Check your connection and try again."
        onRetry={() => {
          if (profile.state.status === "error") profile.reload();
          if (state.status === "error") reload();
        }}
      />
    );
  }
  if (profile.state.status !== "ready" || state.status !== "ready" || needsProfile || completed) {
    return <LoadingState label={completed || needsProfile ? "Continuing…" : "Loading…"} />;
  }
  return <OnboardingSteps key={state.userId} userId={state.userId} initial={state.onboarding} />;
}

/** The step at index i, clamped to the first and last step. */
function stepAt(i: number): OnboardingStep {
  return STEP_ORDER[Math.min(Math.max(i, 0), STEP_ORDER.length - 1)] ?? "goal";
}

function OnboardingSteps({ userId, initial }: { userId: string; initial: Onboarding }) {
  const router = useRouter();
  const [data, setData] = useState(initial);
  // Resume at the first unsaved step; with every step saved, at the last one (Finish).
  const [step, setStep] = useState<OnboardingStep>(initial.next_step ?? "background");
  const [moved, setMoved] = useState(false);

  function go(next: OnboardingStep) {
    setStep(next);
    setMoved(true);
  }

  function stored(next: Onboarding) {
    setData(next);
    setOnboarding(userId, next);
  }

  async function run(step: OnboardingStep, input: unknown): Promise<SaveFailure | null> {
    try {
      if (step === "goal") stored(await saveGoal(input as GoalInput));
      if (step === "schedule") stored(await saveSchedule(input as ScheduleInput));
      if (step === "background") {
        stored(await saveBackground(input as BackgroundInput));
        // Completion makes the page redirect (AFTER_ONBOARDING).
        stored(await completeOnboarding());
        router.replace(AFTER_ONBOARDING);
        return null;
      }
      go(stepAt(STEP_ORDER.indexOf(step) + 1));
      return null;
    } catch (err) {
      if (
        err instanceof ApiError &&
        err.code === "CONFLICT" &&
        err.details?.reason === "onboarding_incomplete"
      ) {
        const missing = err.details.missing_steps as OnboardingStep[] | undefined;
        if (missing?.[0]) go(missing[0]);
        return { kind: "message", message: "Please answer this step first." };
      }
      const failure = saveFailure(err);
      if (failure.kind === "profile") router.replace(PROFILE_SETUP_ROUTE);
      return failure;
    }
  }
  const save = run as StepSave;

  const back = () => go(stepAt(STEP_ORDER.indexOf(step) - 1));

  return (
    <Card className="flex flex-col gap-6 p-6 shadow-sm sm:p-8">
      <OnboardingProgress current={step} />
      {step === "goal" ? (
        <GoalStep key="goal" initial={data.goal} save={save} focusHeading={moved} />
      ) : step === "schedule" ? (
        <ScheduleStep
          key="schedule"
          initial={data.schedule}
          save={save}
          focusHeading={moved}
          onBack={back}
        />
      ) : (
        <BackgroundStep
          key="background"
          initial={data.background}
          save={save}
          focusHeading={moved}
          onBack={back}
        />
      )}
    </Card>
  );
}
