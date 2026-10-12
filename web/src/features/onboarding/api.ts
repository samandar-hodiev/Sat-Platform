// Onboarding API (session 2.02). Contract: docs/API.md → Onboarding.

import { api } from "@/lib/api/client";

import { withAuth } from "@/features/auth/session";

export type OnboardingStep = "goal" | "schedule" | "background";

export type SatReason =
  "college_admission" | "scholarship" | "improve_score" | "practice" | "other";

export interface OnboardingGoal {
  target_score: number;
  /** YYYY-MM-DD; null: the learner has not decided yet. */
  exam_date: string | null;
  /** Days from the learner's today (their time zone); null without an exam date. */
  days_until_exam: number | null;
  reason: SatReason | null;
  saved_at: string;
}

export interface OnboardingSchedule {
  daily_minutes: number;
  /** ISO weekdays, 1 = Monday … 7 = Sunday, sorted. */
  study_days: number[];
  saved_at: string;
}

export interface PreviousScore {
  total: number;
  math: number | null;
  reading_writing: number | null;
  /** YYYY-MM. */
  taken_month: string | null;
}

export interface OnboardingBackground {
  /** 1 (not confident) … 5 (very confident). */
  math_confidence: number;
  reading_writing_confidence: number;
  previous_score: PreviousScore | null;
  saved_at: string;
}

/** A step is null until the learner saves it ("not set", never a default). */
export interface Onboarding {
  completed: boolean;
  completed_at: string | null;
  /** First step not saved yet; null when every step is saved. */
  next_step: OnboardingStep | null;
  goal: OnboardingGoal | null;
  schedule: OnboardingSchedule | null;
  background: OnboardingBackground | null;
}

export interface GoalInput {
  target_score: number;
  exam_date: string | null;
  reason: SatReason | null;
}

export interface ScheduleInput {
  daily_minutes: number;
  study_days: number[];
}

export interface BackgroundInput {
  math_confidence: number;
  reading_writing_confidence: number;
  previous_score: PreviousScore | null;
}

export function fetchOnboarding(): Promise<Onboarding> {
  return withAuth((token) => api().request<Onboarding>("/me/onboarding", { accessToken: token }));
}

function put<T>(step: OnboardingStep, body: T): Promise<Onboarding> {
  return withAuth((token) =>
    api().request<Onboarding>(`/me/onboarding/${step}`, {
      method: "PUT",
      body,
      accessToken: token,
    }),
  );
}

export const saveGoal = (input: GoalInput) => put("goal", input);
export const saveSchedule = (input: ScheduleInput) => put("schedule", input);
export const saveBackground = (input: BackgroundInput) => put("background", input);

export function completeOnboarding(): Promise<Onboarding> {
  return withAuth((token) =>
    api().request<Onboarding>("/me/onboarding/complete", { method: "POST", accessToken: token }),
  );
}
