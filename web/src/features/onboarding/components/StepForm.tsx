"use client";

import { useEffect, useRef, type FormEvent, type ReactNode } from "react";

import { Button } from "@/components/ui";
import { FormAlert } from "@/features/auth/components/FormAlert";
import { ApiError } from "@/lib/api/client";

import { fieldMessage } from "@/features/profile/validation";

import type { BackgroundInput, GoalInput, ScheduleInput } from "../api";

/** One onboarding step: heading, fields, and Back / Continue. */
export function StepForm({
  title,
  description,
  formError,
  submitting,
  submitLabel = "Continue",
  onBack,
  onSubmit,
  focusHeading = false,
  children,
}: {
  title: string;
  description: string;
  formError?: string;
  submitting: boolean;
  submitLabel?: string;
  onBack?: () => void;
  onSubmit: () => void;
  /** Move focus to the heading on mount (after Back / Continue), so screen readers follow. */
  focusHeading?: boolean;
  children: ReactNode;
}) {
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => {
    if (focusHeading) heading.current?.focus();
    // Mount only: a new step is a new StepForm.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!submitting) onSubmit();
  }
  return (
    <form noValidate onSubmit={submit} aria-label={title} className="flex flex-col gap-6">
      <div className="flex flex-col gap-1">
        <h1
          ref={heading}
          tabIndex={-1}
          className="text-2xl font-semibold tracking-tight text-fg outline-none"
        >
          {title}
        </h1>
        <p className="text-sm text-fg-muted">{description}</p>
      </div>
      {formError ? <FormAlert>{formError}</FormAlert> : null}
      <div className="flex flex-col gap-6">{children}</div>
      <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-between">
        {onBack ? (
          <Button type="button" variant="outline" size="lg" onClick={onBack} disabled={submitting}>
            Back
          </Button>
        ) : (
          <span />
        )}
        <Button type="submit" size="lg" loading={submitting} className="sm:min-w-36">
          {submitLabel}
        </Button>
      </div>
    </form>
  );
}

/** A small "Skip" button for an optional question; it clears the answer. */
export function SkipButton({ onClick, label }: { onClick: () => void; label: string }) {
  return (
    <Button type="button" variant="ghost" size="sm" onClick={onClick} aria-label={`Skip: ${label}`}>
      Skip
    </Button>
  );
}

export type SaveFailure =
  | { kind: "profile" }
  | { kind: "fields"; fields: Record<string, string | undefined> }
  | { kind: "message"; message: string };

/** What a failed step save means for the form. */
export function saveFailure(err: unknown): SaveFailure {
  if (err instanceof ApiError) {
    if (err.code === "CONFLICT" && err.details?.reason === "profile_setup_required") {
      return { kind: "profile" };
    }
    if (Object.keys(err.fieldErrors).length > 0) {
      const fields: Record<string, string | undefined> = {};
      for (const [k, v] of Object.entries(err.fieldErrors)) fields[k] = fieldMessage(v);
      return { kind: "fields", fields };
    }
    if (err.code === "NETWORK_ERROR" || err.code === "CONFLICT") {
      return { kind: "message", message: err.message };
    }
  }
  return { kind: "message", message: "Your answers could not be saved. Please try again." };
}

/** Saves a step and moves on; resolves to the failure to show, or null on success. */
export interface StepSave {
  (step: "goal", input: GoalInput): Promise<SaveFailure | null>;
  (step: "schedule", input: ScheduleInput): Promise<SaveFailure | null>;
  (step: "background", input: BackgroundInput): Promise<SaveFailure | null>;
}
