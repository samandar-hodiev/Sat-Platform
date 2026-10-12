"use client";

import { useState } from "react";

import { FormField, Input } from "@/components/ui";

import type { OnboardingBackground } from "../api";
import {
  CONFIDENCE_LEVELS,
  EMPTY_PREVIOUS,
  FIRST_SCORE_MONTH,
  SCORE_MAX,
  SCORE_MIN,
  SECTION_MAX,
  SECTION_MIN,
  hasErrors,
  previousScoreForApi,
  previousScoreForm,
  thisMonth,
  validateConfidence,
  validatePreviousScore,
  type PreviousScoreErrors,
  type PreviousScoreForm,
} from "../validation";
import { RadioChips } from "./ChoiceChips";
import { SkipButton, StepForm, type StepSave } from "./StepForm";

type Errors = {
  math_confidence?: string;
  reading_writing_confidence?: string;
};

const confidenceOptions = CONFIDENCE_LEVELS.map((c) => ({
  value: c.value,
  label: String(c.value),
  ariaLabel: `${c.value}: ${c.label}`,
}));

const TAKEN_OPTIONS: { value: "yes" | "no"; label: string }[] = [
  { value: "yes", label: "Yes" },
  { value: "no", label: "No, not yet" },
];

/** Step 3 (PRD §8.2): confidence in Math and R&W, an earlier SAT result (optional). */
export function BackgroundStep({
  initial,
  save,
  focusHeading,
  onBack,
}: {
  initial: OnboardingBackground | null;
  save: StepSave;
  focusHeading?: boolean;
  onBack: () => void;
}) {
  const [math, setMath] = useState<number | null>(initial?.math_confidence ?? null);
  const [rw, setRw] = useState<number | null>(initial?.reading_writing_confidence ?? null);
  const [taken, setTaken] = useState<"yes" | "no" | null>(
    initial ? (initial.previous_score ? "yes" : "no") : null,
  );
  const [prev, setPrev] = useState<PreviousScoreForm>(previousScoreForm(initial?.previous_score));
  const [errors, setErrors] = useState<Errors>({});
  const [prevErrors, setPrevErrors] = useState<PreviousScoreErrors>({});
  const [formError, setFormError] = useState<string>();
  const [submitting, setSubmitting] = useState(false);

  const set = (k: keyof PreviousScoreForm) => (e: { target: { value: string } }) => {
    setPrev((p) => ({ ...p, [k]: e.target.value }));
    // The total's "must add up" error depends on the sections too.
    setPrevErrors((x) => ({
      ...x,
      [k]: undefined,
      ...(k === "takenMonth" ? {} : { total: undefined }),
    }));
  };
  const clear = (field: keyof Errors) => setErrors((e) => ({ ...e, [field]: undefined }));

  function skipPrevious() {
    setTaken(null);
    setPrev(EMPTY_PREVIOUS);
    setPrevErrors({});
  }

  async function submit() {
    const found: Errors = {
      math_confidence: validateConfidence(math),
      reading_writing_confidence: validateConfidence(rw),
    };
    const foundPrev = taken === "yes" ? validatePreviousScore(prev) : {};
    setErrors(found);
    setPrevErrors(foundPrev);
    setFormError(undefined);
    if (hasErrors(found) || hasErrors(foundPrev) || math === null || rw === null) return;
    setSubmitting(true);
    const failure = await save("background", {
      math_confidence: math,
      reading_writing_confidence: rw,
      previous_score: taken === "yes" ? previousScoreForApi(prev) : null,
    });
    setSubmitting(false);
    if (failure?.kind === "fields") {
      const f = failure.fields;
      setErrors({
        math_confidence: f.math_confidence,
        reading_writing_confidence: f.reading_writing_confidence,
      });
      setPrevErrors({
        total: f["previous_score.total"],
        math: f["previous_score.math"],
        readingWriting: f["previous_score.reading_writing"],
        takenMonth: f["previous_score.taken_month"],
      });
    }
    if (failure?.kind === "message") setFormError(failure.message);
  }

  return (
    <StepForm
      title="Where are you now?"
      description="A quick self-check, so your plan starts at the right level. There are no wrong answers."
      formError={formError}
      submitting={submitting}
      focusHeading={focusHeading}
      submitLabel="Finish"
      onBack={onBack}
      onSubmit={submit}
    >
      <RadioChips<number>
        legend="How confident are you in Math?"
        required
        hint="1 = not confident, 5 = very confident."
        error={errors.math_confidence}
        options={confidenceOptions}
        value={math}
        onChange={(v) => {
          setMath(v);
          clear("math_confidence");
        }}
        columns="grid-cols-5"
      />
      <RadioChips<number>
        legend="How confident are you in Reading and Writing?"
        required
        hint="1 = not confident, 5 = very confident."
        error={errors.reading_writing_confidence}
        options={confidenceOptions}
        value={rw}
        onChange={(v) => {
          setRw(v);
          clear("reading_writing_confidence");
        }}
        columns="grid-cols-5"
      />

      <div className="flex flex-col gap-4">
        <RadioChips<"yes" | "no">
          legend="Have you taken the official SAT before?"
          hint="Optional."
          options={TAKEN_OPTIONS}
          value={taken}
          onChange={setTaken}
          columns="grid-cols-2"
          action={
            <SkipButton label="Have you taken the official SAT before?" onClick={skipPrevious} />
          }
        />
        {taken === "yes" ? (
          <div className="flex flex-col gap-4 rounded-lg border border-border bg-surface-muted p-4">
            <FormField
              label="Total score"
              required
              hint={`${SCORE_MIN}–${SCORE_MAX}.`}
              error={prevErrors.total}
            >
              <Input
                type="number"
                inputMode="numeric"
                min={SCORE_MIN}
                max={SCORE_MAX}
                step={10}
                value={prev.total}
                onChange={set("total")}
              />
            </FormField>
            <div className="grid gap-4 sm:grid-cols-2">
              <FormField
                label="Math"
                hint={`Optional, ${SECTION_MIN}–${SECTION_MAX}.`}
                error={prevErrors.math}
              >
                <Input
                  type="number"
                  inputMode="numeric"
                  min={SECTION_MIN}
                  max={SECTION_MAX}
                  step={10}
                  value={prev.math}
                  onChange={set("math")}
                />
              </FormField>
              <FormField
                label="Reading and Writing"
                hint={`Optional, ${SECTION_MIN}–${SECTION_MAX}.`}
                error={prevErrors.readingWriting}
              >
                <Input
                  type="number"
                  inputMode="numeric"
                  min={SECTION_MIN}
                  max={SECTION_MAX}
                  step={10}
                  value={prev.readingWriting}
                  onChange={set("readingWriting")}
                />
              </FormField>
            </div>
            <FormField
              label="When did you take it?"
              hint="Optional. The month is enough, even if approximate."
              error={prevErrors.takenMonth}
            >
              <Input
                type="month"
                min={FIRST_SCORE_MONTH}
                max={thisMonth()}
                placeholder="YYYY-MM"
                value={prev.takenMonth}
                onChange={set("takenMonth")}
              />
            </FormField>
          </div>
        ) : null}
      </div>
    </StepForm>
  );
}
