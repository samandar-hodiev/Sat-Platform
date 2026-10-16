"use client";

import { useState } from "react";

import {
  Button,
  Checkbox,
  ErrorState,
  FormField,
  Input,
  Skeleton,
  useToast,
} from "@/components/ui";

import {
  saveGoal,
  saveSchedule,
  type OnboardingGoal,
  type OnboardingSchedule,
  type SatReason,
} from "@/features/onboarding/api";
import { CheckboxChips, RadioChips } from "@/features/onboarding/components/ChoiceChips";
import { saveFailure } from "@/features/onboarding/components/StepForm";
import { setOnboarding, useOnboarding } from "@/features/onboarding";
import {
  DAILY_MINUTES,
  REASONS,
  SCORE_MAX,
  SCORE_MIN,
  WEEKDAYS,
  hasErrors,
  latestExamDate,
  parseScore,
  todayISO,
  validateDailyMinutes,
  validateExamDate,
  validateStudyDays,
  validateTargetScore,
} from "@/features/onboarding/validation";

import { Section } from "./Section";

/**
 * Settings → Learning preferences (PRD §20): the onboarding answers (2.02), edited through the
 * same endpoints and rules. The study plan (2.05) reads them; it does not exist yet, so nothing
 * is recalculated here.
 */
export function LearningPreferences() {
  const { state, reload } = useOnboarding();

  if (state.status === "error") {
    return (
      <ErrorState
        title="Learning preferences could not be loaded"
        description="Check your connection and try again."
        onRetry={reload}
      />
    );
  }
  if (state.status !== "ready") {
    return (
      <div
        className="flex flex-col gap-6"
        aria-busy="true"
        aria-label="Loading learning preferences"
      >
        <Skeleton className="h-72 w-full rounded-lg" />
        <Skeleton className="h-56 w-full rounded-lg" />
      </div>
    );
  }
  const { userId, onboarding } = state;
  return (
    <div className="flex flex-col gap-6">
      <GoalForm userId={userId} initial={onboarding.goal} />
      <ScheduleForm userId={userId} initial={onboarding.schedule} />
    </div>
  );
}

type GoalErrors = { target_score?: string; exam_date?: string; reason?: string };

function GoalForm({ userId, initial }: { userId: string; initial: OnboardingGoal | null }) {
  const { toast } = useToast();
  const [target, setTarget] = useState(initial ? String(initial.target_score) : "");
  const [examDate, setExamDate] = useState(initial?.exam_date ?? "");
  const [undecided, setUndecided] = useState(initial !== null && initial.exam_date === null);
  const [reason, setReason] = useState<SatReason | null>(initial?.reason ?? null);
  const [errors, setErrors] = useState<GoalErrors>({});
  const [formError, setFormError] = useState<string>();
  const [saving, setSaving] = useState(false);
  const clear = (field: keyof GoalErrors) => setErrors((e) => ({ ...e, [field]: undefined }));

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    const found: GoalErrors = {
      target_score: validateTargetScore(target),
      exam_date: validateExamDate(examDate, undecided),
    };
    setErrors(found);
    setFormError(undefined);
    if (hasErrors(found)) return;
    setSaving(true);
    try {
      const next = await saveGoal({
        target_score: parseScore(target),
        exam_date: undecided ? null : examDate,
        reason,
      });
      setOnboarding(userId, next);
      toast({ variant: "success", title: "Goal saved" });
    } catch (err) {
      const failure = saveFailure(err);
      if (failure.kind === "fields") setErrors(failure.fields);
      else
        setFormError(
          failure.kind === "message"
            ? failure.message
            : "Your goal could not be saved. Please try again.",
        );
    } finally {
      setSaving(false);
    }
  }

  return (
    <Section title="SAT goal" description="Your target score and exam date.">
      <form noValidate onSubmit={submit} className="flex flex-col gap-5">
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
          action={
            reason ? (
              <Button type="button" variant="ghost" size="sm" onClick={() => setReason(null)}>
                Clear
              </Button>
            ) : null
          }
        />
        <FormFooter error={formError} saving={saving} label="Save goal" />
      </form>
    </Section>
  );
}

type ScheduleErrors = { daily_minutes?: string; study_days?: string };

function ScheduleForm({ userId, initial }: { userId: string; initial: OnboardingSchedule | null }) {
  const { toast } = useToast();
  const [minutes, setMinutes] = useState<number | null>(initial?.daily_minutes ?? null);
  const [days, setDays] = useState<number[]>(initial?.study_days ?? []);
  const [errors, setErrors] = useState<ScheduleErrors>({});
  const [formError, setFormError] = useState<string>();
  const [saving, setSaving] = useState(false);
  const clear = (field: keyof ScheduleErrors) => setErrors((e) => ({ ...e, [field]: undefined }));

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    const found: ScheduleErrors = {
      daily_minutes: validateDailyMinutes(minutes),
      study_days: validateStudyDays(days),
    };
    setErrors(found);
    setFormError(undefined);
    if (hasErrors(found) || minutes === null) return;
    setSaving(true);
    try {
      const next = await saveSchedule({ daily_minutes: minutes, study_days: days });
      setOnboarding(userId, next);
      toast({ variant: "success", title: "Study time saved" });
    } catch (err) {
      const failure = saveFailure(err);
      if (failure.kind === "fields") setErrors(failure.fields);
      else
        setFormError(
          failure.kind === "message"
            ? failure.message
            : "Your study time could not be saved. Please try again.",
        );
    } finally {
      setSaving(false);
    }
  }

  return (
    <Section title="Study time" description="How much and on which days you plan to study.">
      <form noValidate onSubmit={submit} className="flex flex-col gap-5">
        <RadioChips<number>
          legend="Daily study time"
          required
          error={errors.daily_minutes}
          options={DAILY_MINUTES}
          value={minutes}
          onChange={(m) => {
            setMinutes(m);
            clear("daily_minutes");
          }}
          columns="grid-cols-3 sm:grid-cols-6"
        />
        <CheckboxChips<number>
          legend="Study days"
          required
          error={errors.study_days}
          options={WEEKDAYS.map((d) => ({ value: d.value, label: d.short, ariaLabel: d.label }))}
          values={days}
          onChange={(d) => {
            setDays(d);
            clear("study_days");
          }}
          columns="grid-cols-4 sm:grid-cols-7"
        />
        <FormFooter error={formError} saving={saving} label="Save study time" />
      </form>
    </Section>
  );
}

function FormFooter({ error, saving, label }: { error?: string; saving: boolean; label: string }) {
  return (
    <div className="flex flex-col gap-2 border-t border-border pt-4 sm:flex-row sm:items-center sm:justify-between">
      {error ? (
        <p role="alert" className="text-sm text-error-text">
          {error}
        </p>
      ) : (
        <span />
      )}
      <Button type="submit" loading={saving}>
        {label}
      </Button>
    </div>
  );
}
