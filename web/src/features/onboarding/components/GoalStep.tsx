"use client";

import { useState } from "react";

import { Checkbox, FormField, Input } from "@/components/ui";

import type { OnboardingGoal, SatReason } from "../api";
import {
  REASONS,
  SCORE_MAX,
  SCORE_MIN,
  hasErrors,
  latestExamDate,
  parseScore,
  todayISO,
  validateExamDate,
  validateTargetScore,
} from "../validation";
import { RadioChips } from "./ChoiceChips";
import { SkipButton, StepForm, type SaveFailure, type StepSave } from "./StepForm";

type Errors = { target_score?: string; exam_date?: string; reason?: string };

/** Step 1 (PRD §8.1): target score, exam date, reason (optional). */
export function GoalStep({
  initial,
  save,
  focusHeading,
}: {
  initial: OnboardingGoal | null;
  save: StepSave;
  focusHeading?: boolean;
}) {
  const [target, setTarget] = useState(initial ? String(initial.target_score) : "");
  const [examDate, setExamDate] = useState(initial?.exam_date ?? "");
  const [undecided, setUndecided] = useState(initial !== null && initial.exam_date === null);
  const [reason, setReason] = useState<SatReason | null>(initial?.reason ?? null);
  const [errors, setErrors] = useState<Errors>({});
  const [formError, setFormError] = useState<string>();
  const [submitting, setSubmitting] = useState(false);
  // An answer that changes is no longer the one the error was about.
  const clear = (field: keyof Errors) => setErrors((e) => ({ ...e, [field]: undefined }));

  async function submit() {
    const found: Errors = {
      target_score: validateTargetScore(target),
      exam_date: validateExamDate(examDate, undecided),
    };
    setErrors(found);
    setFormError(undefined);
    if (hasErrors(found)) return;
    setSubmitting(true);
    const failure: SaveFailure | null = await save("goal", {
      target_score: parseScore(target),
      exam_date: undecided ? null : examDate,
      reason,
    });
    setSubmitting(false);
    if (failure?.kind === "fields") setErrors(failure.fields);
    if (failure?.kind === "message") setFormError(failure.message);
  }

  return (
    <StepForm
      title="Set your SAT goal"
      description="We use your goal to plan what to study and how much. You can change it later."
      formError={formError}
      submitting={submitting}
      focusHeading={focusHeading}
      onSubmit={submit}
    >
      <FormField
        label="Target score"
        required
        hint={`Total score from ${SCORE_MIN} to ${SCORE_MAX}, in steps of 10.`}
        error={errors.target_score}
      >
        <Input
          type="number"
          inputMode="numeric"
          min={SCORE_MIN}
          max={SCORE_MAX}
          step={10}
          placeholder="e.g. 1400"
          value={target}
          onChange={(e) => {
            setTarget(e.target.value);
            clear("target_score");
          }}
        />
      </FormField>

      <div className="flex flex-col gap-3">
        <FormField label="Exam date" required={!undecided} error={errors.exam_date}>
          <Input
            type="date"
            min={todayISO()}
            max={latestExamDate()}
            value={undecided ? "" : examDate}
            disabled={undecided}
            onChange={(e) => {
              setExamDate(e.target.value);
              clear("exam_date");
            }}
          />
        </FormField>
        <Checkbox
          label="I haven't decided yet"
          checked={undecided}
          onChange={(e) => {
            setUndecided(e.target.checked);
            clear("exam_date");
          }}
        />
      </div>

      <RadioChips<SatReason>
        legend="Why are you taking the SAT?"
        hint="Optional."
        error={errors.reason}
        options={REASONS}
        value={reason}
        onChange={(r) => {
          setReason(r);
          clear("reason");
        }}
        columns="grid-cols-1 sm:grid-cols-2"
        action={<SkipButton label="Why are you taking the SAT?" onClick={() => setReason(null)} />}
      />
    </StepForm>
  );
}
