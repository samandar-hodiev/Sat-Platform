"use client";

import { useState } from "react";

import type { OnboardingSchedule } from "../api";
import {
  DAILY_MINUTES,
  WEEKDAYS,
  hasErrors,
  validateDailyMinutes,
  validateStudyDays,
} from "../validation";
import { CheckboxChips, RadioChips } from "./ChoiceChips";
import { StepForm, type StepSave } from "./StepForm";

type Errors = { daily_minutes?: string; study_days?: string };

/** Step 2 (PRD §8.2): daily study time and study days. */
export function ScheduleStep({
  initial,
  save,
  focusHeading,
  onBack,
}: {
  initial: OnboardingSchedule | null;
  save: StepSave;
  focusHeading?: boolean;
  onBack: () => void;
}) {
  const [minutes, setMinutes] = useState<number | null>(initial?.daily_minutes ?? null);
  const [days, setDays] = useState<number[]>(initial?.study_days ?? []);
  const [errors, setErrors] = useState<Errors>({});
  const [formError, setFormError] = useState<string>();
  const [submitting, setSubmitting] = useState(false);
  const clear = (field: keyof Errors) => setErrors((e) => ({ ...e, [field]: undefined }));

  async function submit() {
    const found: Errors = {
      daily_minutes: validateDailyMinutes(minutes),
      study_days: validateStudyDays(days),
    };
    setErrors(found);
    setFormError(undefined);
    if (hasErrors(found) || minutes === null) return;
    setSubmitting(true);
    const failure = await save("schedule", { daily_minutes: minutes, study_days: days });
    setSubmitting(false);
    if (failure?.kind === "fields") setErrors(failure.fields);
    if (failure?.kind === "message") setFormError(failure.message);
  }

  return (
    <StepForm
      title="Plan your study time"
      description="Be realistic: a steady routine beats long, rare sessions."
      formError={formError}
      submitting={submitting}
      focusHeading={focusHeading}
      onBack={onBack}
      onSubmit={submit}
    >
      <RadioChips<number>
        legend="How much time can you study a day?"
        required
        error={errors.daily_minutes}
        options={DAILY_MINUTES}
        value={minutes}
        onChange={(m) => {
          setMinutes(m);
          clear("daily_minutes");
        }}
        columns="grid-cols-3"
      />
      <CheckboxChips<number>
        legend="Which days can you study?"
        required
        hint="Choose every day that usually works for you."
        error={errors.study_days}
        options={WEEKDAYS.map((d) => ({ value: d.value, label: d.short, ariaLabel: d.label }))}
        values={days}
        onChange={(d) => {
          setDays(d);
          clear("study_days");
        }}
        columns="grid-cols-4 sm:grid-cols-7"
      />
    </StepForm>
  );
}
