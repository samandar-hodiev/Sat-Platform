// Onboarding answer rules (session 2.02). The API checks the same rules
// (api/internal/onboarding/validate.go); these give instant feedback.

import type { PreviousScore, SatReason } from "./api";

export const SCORE_MIN = 400;
export const SCORE_MAX = 1600;
export const SECTION_MIN = 200;
export const SECTION_MAX = 800;
export const EXAM_DATE_MAX_YEARS = 2;
/** The 1600-point SAT started in March 2016. */
export const FIRST_SCORE_MONTH = "2016-03";

export const REASONS: { value: SatReason; label: string }[] = [
  { value: "college_admission", label: "College admission" },
  { value: "scholarship", label: "Scholarship" },
  { value: "improve_score", label: "Improve my score" },
  { value: "practice", label: "Practice and learning" },
  { value: "other", label: "Other" },
];

export const DAILY_MINUTES: { value: number; label: string }[] = [
  { value: 15, label: "15 min" },
  { value: 30, label: "30 min" },
  { value: 45, label: "45 min" },
  { value: 60, label: "1 hour" },
  { value: 90, label: "1.5 hours" },
  { value: 120, label: "2+ hours" },
];

/** ISO weekdays: 1 = Monday … 7 = Sunday. */
export const WEEKDAYS: { value: number; short: string; label: string }[] = [
  { value: 1, short: "Mon", label: "Monday" },
  { value: 2, short: "Tue", label: "Tuesday" },
  { value: 3, short: "Wed", label: "Wednesday" },
  { value: 4, short: "Thu", label: "Thursday" },
  { value: 5, short: "Fri", label: "Friday" },
  { value: 6, short: "Sat", label: "Saturday" },
  { value: 7, short: "Sun", label: "Sunday" },
];

export const CONFIDENCE_LEVELS: { value: number; label: string }[] = [
  { value: 1, label: "Not confident" },
  { value: 2, label: "A little" },
  { value: 3, label: "Somewhat" },
  { value: 4, label: "Confident" },
  { value: 5, label: "Very confident" },
];

function pad(n: number): string {
  return String(n).padStart(2, "0");
}

/** YYYY-MM-DD of the device's today. */
export function todayISO(now: Date = new Date()): string {
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`;
}

/** The latest exam date allowed: today + EXAM_DATE_MAX_YEARS. */
export function latestExamDate(now: Date = new Date()): string {
  return `${now.getFullYear() + EXAM_DATE_MAX_YEARS}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`;
}

/** YYYY-MM of the device's current month. */
export function thisMonth(now: Date = new Date()): string {
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}`;
}

/** A whole score from the input text, or NaN. */
export function parseScore(raw: string): number {
  const s = raw.trim();
  return /^\d{1,4}$/.test(s) ? Number(s) : Number.NaN;
}

function scoreError(raw: string, min: number, max: number, required: boolean): string | undefined {
  if (raw.trim() === "") return required ? "Enter a score." : undefined;
  const v = parseScore(raw);
  if (Number.isNaN(v) || v < min || v > max || v % 10 !== 0) {
    return `Enter a score from ${min} to ${max}, in steps of 10.`;
  }
  return undefined;
}

export function validateTargetScore(raw: string): string | undefined {
  return scoreError(raw, SCORE_MIN, SCORE_MAX, true);
}

/** An empty date is valid only when the learner marked it undecided. */
export function validateExamDate(
  date: string,
  undecided: boolean,
  now: Date = new Date(),
): string | undefined {
  if (undecided) return undefined;
  if (!date) return "Choose your exam date, or tick “I haven't decided yet”.";
  if (!/^\d{4}-\d{2}-\d{2}$/.test(date)) return "Enter a real date.";
  if (date < todayISO(now)) return "The exam date can't be in the past.";
  if (date > latestExamDate(now)) return "Choose a date within the next 2 years.";
  return undefined;
}

export function validateDailyMinutes(v: number | null): string | undefined {
  return v === null ? "Choose how much time you can study a day." : undefined;
}

export function validateStudyDays(days: number[]): string | undefined {
  return days.length === 0 ? "Choose at least one day." : undefined;
}

export function validateConfidence(v: number | null): string | undefined {
  return v === null ? "Choose one." : undefined;
}

export interface PreviousScoreForm {
  total: string;
  math: string;
  readingWriting: string;
  /** YYYY-MM or "". */
  takenMonth: string;
}

export const EMPTY_PREVIOUS: PreviousScoreForm = {
  total: "",
  math: "",
  readingWriting: "",
  takenMonth: "",
};

export type PreviousScoreErrors = Partial<Record<keyof PreviousScoreForm, string>>;

/** Sections are optional, but come together and add up to the total. */
export function validatePreviousScore(
  f: PreviousScoreForm,
  now: Date = new Date(),
): PreviousScoreErrors {
  const e: PreviousScoreErrors = {
    total: scoreError(f.total, SCORE_MIN, SCORE_MAX, true),
    math: scoreError(f.math, SECTION_MIN, SECTION_MAX, false),
    readingWriting: scoreError(f.readingWriting, SECTION_MIN, SECTION_MAX, false),
  };
  const hasMath = f.math.trim() !== "";
  const hasRW = f.readingWriting.trim() !== "";
  if (hasMath && !hasRW) e.readingWriting = "Enter both section scores, or neither.";
  if (hasRW && !hasMath) e.math = "Enter both section scores, or neither.";
  if (hasMath && hasRW && !e.total && !e.math && !e.readingWriting) {
    if (parseScore(f.math) + parseScore(f.readingWriting) !== parseScore(f.total)) {
      e.total = "The total must equal Math plus Reading and Writing.";
    }
  }
  if (f.takenMonth) {
    if (!/^\d{4}-\d{2}$/.test(f.takenMonth)) e.takenMonth = "Enter a month, e.g. 2026-05.";
    else if (f.takenMonth < FIRST_SCORE_MONTH) e.takenMonth = "Enter March 2016 or later.";
    else if (f.takenMonth > thisMonth(now)) e.takenMonth = "The month can't be in the future.";
  }
  return e;
}

export function previousScoreForApi(f: PreviousScoreForm): PreviousScore {
  const opt = (s: string) => (s.trim() === "" ? null : parseScore(s));
  return {
    total: parseScore(f.total),
    math: opt(f.math),
    reading_writing: opt(f.readingWriting),
    taken_month: f.takenMonth || null,
  };
}

export function previousScoreForm(p: PreviousScore | null | undefined): PreviousScoreForm {
  if (!p) return EMPTY_PREVIOUS;
  return {
    total: String(p.total),
    math: p.math === null ? "" : String(p.math),
    readingWriting: p.reading_writing === null ? "" : String(p.reading_writing),
    takenMonth: p.taken_month ?? "",
  };
}

export function hasErrors(errors: Record<string, string | undefined>): boolean {
  return Object.values(errors).some(Boolean);
}
