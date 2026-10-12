// @vitest-environment jsdom
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "@/lib/api/client";

import * as api from "../api";
import type { Onboarding } from "../api";
import { getOnboardingState, resetOnboardingForTests } from "../onboardingStore";
import { todayISO } from "../validation";
import { OnboardingFlow } from "./OnboardingFlow";
import { OnboardingGate } from "./OnboardingGate";

const replace = vi.fn();
vi.mock("next/navigation", () => ({ useRouter: () => ({ replace, push: vi.fn() }) }));

const sessionUser = {
  id: "u1",
  email: "ali@example.com",
  role: "learner" as const,
  email_verified: true,
  has_password: true,
  created_at: "2026-10-12T00:00:00Z",
};
vi.mock("@/features/auth/session", () => ({
  useSession: () => ({ status: "signed-in", user: sessionUser }),
}));

const profileState = vi.hoisted(() => ({ completed: true }));
vi.mock("@/features/profile", () => ({
  useLearnerProfile: () => ({
    state: {
      status: "ready",
      userId: "u1",
      profile: { profile_completed: profileState.completed },
    },
    reload: vi.fn(),
  }),
}));

vi.mock("../api", () => ({
  fetchOnboarding: vi.fn(),
  saveGoal: vi.fn(),
  saveSchedule: vi.fn(),
  saveBackground: vi.fn(),
  completeOnboarding: vi.fn(),
}));
const mocked = vi.mocked(api);

const SAVED = "2026-10-12T10:00:00Z";

function onboarding(over: Partial<Onboarding> = {}): Onboarding {
  return {
    completed: false,
    completed_at: null,
    next_step: "goal",
    goal: null,
    schedule: null,
    background: null,
    ...over,
  };
}

const goal = {
  target_score: 1450,
  exam_date: null,
  days_until_exam: null,
  reason: null,
  saved_at: SAVED,
};
const schedule = { daily_minutes: 45, study_days: [1, 3], saved_at: SAVED };
const background = {
  math_confidence: 3,
  reading_writing_confidence: 4,
  previous_score: null,
  saved_at: SAVED,
};

/** A date a month from today, as the date input expects. */
function nextMonth(): string {
  const d = new Date();
  d.setMonth(d.getMonth() + 1);
  return todayISO(d);
}

/** A 1–5 choice inside the Math or Reading and Writing confidence group. */
function confidence(section: "Math" | "Reading and Writing", option: string) {
  const group = screen.getByRole("group", { name: new RegExp(`confident are you in ${section}`) });
  return within(group).getByRole("radio", { name: option });
}

beforeEach(() => {
  vi.resetAllMocks();
  resetOnboardingForTests();
  profileState.completed = true;
});

describe("OnboardingFlow", () => {
  it("starts at the goal step with a progress indicator", async () => {
    mocked.fetchOnboarding.mockResolvedValue(onboarding());
    render(<OnboardingFlow />);
    expect(await screen.findByRole("heading", { name: "Set your SAT goal" })).toBeTruthy();
    const progress = screen.getByRole("navigation", { name: "Onboarding progress" });
    expect(within(progress).getByText("Step 1 of 3")).toBeTruthy();
    expect(
      within(progress).getByText("Your goal").closest("li")?.getAttribute("aria-current"),
    ).toBe("step");
    expect(screen.queryByRole("button", { name: "Back" })).toBeNull();
  });

  it("validates the goal before saving", async () => {
    mocked.fetchOnboarding.mockResolvedValue(onboarding());
    render(<OnboardingFlow />);
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: "Continue" }));
    expect(screen.getByText("Enter a score.")).toBeTruthy();
    expect(screen.getByText(/Choose your exam date/)).toBeTruthy();
    await user.type(screen.getByLabelText(/Target score/), "1455");
    // Changing an answer clears its error; the other one stays.
    expect(screen.queryByText("Enter a score.")).toBeNull();
    expect(screen.getByText(/Choose your exam date/)).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Continue" }));
    expect(screen.getByText(/in steps of 10/)).toBeTruthy();
    expect(mocked.saveGoal).not.toHaveBeenCalled();
  });

  it("saves each step, goes back with answers kept, and finishes", async () => {
    mocked.fetchOnboarding.mockResolvedValue(onboarding());
    const exam = nextMonth();
    mocked.saveGoal.mockResolvedValue(
      onboarding({
        next_step: "schedule",
        goal: { ...goal, exam_date: exam, reason: "scholarship" },
      }),
    );
    mocked.saveSchedule.mockResolvedValue(
      onboarding({ next_step: "background", goal: { ...goal, exam_date: exam }, schedule }),
    );
    mocked.saveBackground.mockResolvedValue(
      onboarding({ next_step: null, goal, schedule, background }),
    );
    mocked.completeOnboarding.mockResolvedValue(
      onboarding({
        completed: true,
        completed_at: SAVED,
        next_step: null,
        goal,
        schedule,
        background,
      }),
    );

    render(<OnboardingFlow />);
    const user = userEvent.setup();
    await user.type(await screen.findByLabelText(/Target score/), "1450");
    await user.type(screen.getByLabelText(/Exam date/), exam);
    await user.click(screen.getByRole("radio", { name: "Scholarship" }));
    await user.click(screen.getByRole("button", { name: "Continue" }));
    expect(mocked.saveGoal).toHaveBeenCalledWith({
      target_score: 1450,
      exam_date: exam,
      reason: "scholarship",
    });

    // Step 2; focus moves to the new heading.
    const heading = await screen.findByRole("heading", { name: "Plan your study time" });
    expect(document.activeElement).toBe(heading);
    await user.click(screen.getByRole("button", { name: "Continue" }));
    expect(screen.getByText("Choose how much time you can study a day.")).toBeTruthy();
    expect(screen.getByText("Choose at least one day.")).toBeTruthy();
    await user.click(screen.getByRole("radio", { name: "45 min" }));
    await user.click(screen.getByRole("checkbox", { name: "Wednesday" }));
    await user.click(screen.getByRole("checkbox", { name: "Monday" }));
    await user.click(screen.getByRole("button", { name: "Continue" }));
    expect(mocked.saveSchedule).toHaveBeenCalledWith({ daily_minutes: 45, study_days: [1, 3] });

    // Back to step 2 shows the saved answers, then forward again.
    await screen.findByRole("heading", { name: "Where are you now?" });
    await user.click(screen.getByRole("button", { name: "Back" }));
    expect((screen.getByRole("radio", { name: "45 min" }) as HTMLInputElement).checked).toBe(true);
    expect((screen.getByRole("checkbox", { name: "Monday" }) as HTMLInputElement).checked).toBe(
      true,
    );
    await user.click(screen.getByRole("button", { name: "Continue" }));

    await screen.findByRole("heading", { name: "Where are you now?" });
    await user.click(confidence("Math", "3: Somewhat"));
    await user.click(confidence("Reading and Writing", "4: Confident"));
    await user.click(screen.getByRole("button", { name: "Finish" }));

    await waitFor(() => expect(replace).toHaveBeenCalledWith("/app"));
    expect(mocked.saveBackground).toHaveBeenCalledWith({
      math_confidence: 3,
      reading_writing_confidence: 4,
      previous_score: null,
    });
    expect(mocked.completeOnboarding).toHaveBeenCalledTimes(1);
    const s = getOnboardingState();
    expect(s.status === "ready" && s.onboarding.completed).toBe(true);
  });

  it("marks the exam date undecided and skips the optional reason", async () => {
    mocked.fetchOnboarding.mockResolvedValue(onboarding());
    mocked.saveGoal.mockResolvedValue(onboarding({ next_step: "schedule", goal }));
    render(<OnboardingFlow />);
    const user = userEvent.setup();
    await user.type(await screen.findByLabelText(/Target score/), "1450");
    await user.click(screen.getByRole("checkbox", { name: "I haven't decided yet" }));
    expect((screen.getByLabelText(/Exam date/) as HTMLInputElement).disabled).toBe(true);
    await user.click(screen.getByRole("radio", { name: "Other" }));
    await user.click(screen.getByRole("button", { name: "Skip: Why are you taking the SAT?" }));
    expect((screen.getByRole("radio", { name: "Other" }) as HTMLInputElement).checked).toBe(false);
    await user.click(screen.getByRole("button", { name: "Continue" }));
    expect(mocked.saveGoal).toHaveBeenCalledWith({
      target_score: 1450,
      exam_date: null,
      reason: null,
    });
  });

  it("resumes at the first unsaved step with earlier answers prefilled", async () => {
    mocked.fetchOnboarding.mockResolvedValue(
      onboarding({ next_step: "background", goal, schedule }),
    );
    render(<OnboardingFlow />);
    expect(await screen.findByRole("heading", { name: "Where are you now?" })).toBeTruthy();
    expect(screen.getByText("Step 3 of 3")).toBeTruthy();
    // Resuming does not steal focus.
    expect(document.activeElement).toBe(document.body);
    await userEvent.click(screen.getByRole("button", { name: "Back" }));
    expect((screen.getByRole("checkbox", { name: "Wednesday" }) as HTMLInputElement).checked).toBe(
      true,
    );
  });

  it("collects an earlier SAT result and checks it", async () => {
    mocked.fetchOnboarding.mockResolvedValue(
      onboarding({ next_step: "background", goal, schedule }),
    );
    mocked.saveBackground.mockResolvedValue(
      onboarding({ next_step: null, goal, schedule, background }),
    );
    mocked.completeOnboarding.mockResolvedValue(
      onboarding({
        completed: true,
        completed_at: SAVED,
        next_step: null,
        goal,
        schedule,
        background,
      }),
    );
    render(<OnboardingFlow />);
    const user = userEvent.setup();
    await screen.findByRole("heading", { name: "Where are you now?" });
    await user.click(confidence("Math", "2: A little"));
    await user.click(confidence("Reading and Writing", "5: Very confident"));
    await user.click(screen.getByRole("radio", { name: "Yes" }));
    await user.type(screen.getByLabelText(/Total score/), "1200");
    await user.type(screen.getByLabelText(/^Math/), "700");
    await user.type(screen.getByLabelText(/Reading and Writing/), "600");
    await user.click(screen.getByRole("button", { name: "Finish" }));
    expect(screen.getByText(/must equal Math plus/)).toBeTruthy();
    expect(mocked.saveBackground).not.toHaveBeenCalled();

    await user.clear(screen.getByLabelText(/^Math/));
    await user.type(screen.getByLabelText(/^Math/), "600");
    await user.click(screen.getByRole("button", { name: "Finish" }));
    await waitFor(() =>
      expect(mocked.saveBackground).toHaveBeenCalledWith({
        math_confidence: 2,
        reading_writing_confidence: 5,
        previous_score: { total: 1200, math: 600, reading_writing: 600, taken_month: null },
      }),
    );
  });

  it("shows field errors from the API", async () => {
    mocked.fetchOnboarding.mockResolvedValue(onboarding());
    mocked.saveGoal.mockRejectedValue(
      new ApiError(422, {
        code: "VALIDATION_ERROR",
        message: "Some fields are invalid",
        details: { fields: { exam_date: "must be today or later" } },
      }),
    );
    render(<OnboardingFlow />);
    const user = userEvent.setup();
    await user.type(await screen.findByLabelText(/Target score/), "1300");
    await user.type(screen.getByLabelText(/Exam date/), nextMonth());
    await user.click(screen.getByRole("button", { name: "Continue" }));
    expect(await screen.findByText("Must be today or later.")).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Set your SAT goal" })).toBeTruthy();
  });

  it("shows a message when saving fails and keeps the answers", async () => {
    mocked.fetchOnboarding.mockResolvedValue(onboarding({ next_step: "schedule", goal }));
    mocked.saveSchedule.mockRejectedValue(new Error("boom"));
    render(<OnboardingFlow />);
    const user = userEvent.setup();
    await user.click(await screen.findByRole("radio", { name: "1 hour" }));
    await user.click(screen.getByRole("checkbox", { name: "Sunday" }));
    await user.click(screen.getByRole("button", { name: "Continue" }));
    expect((await screen.findByRole("alert")).textContent).toMatch(/could not be saved/);
    expect((screen.getByRole("radio", { name: "1 hour" }) as HTMLInputElement).checked).toBe(true);
  });

  it("goes back to the missing step when finishing is refused", async () => {
    mocked.fetchOnboarding.mockResolvedValue(
      onboarding({ next_step: "background", goal, schedule }),
    );
    mocked.saveBackground.mockResolvedValue(
      onboarding({ next_step: "schedule", goal, background }),
    );
    mocked.completeOnboarding.mockRejectedValue(
      new ApiError(409, {
        code: "CONFLICT",
        message: "Answer every onboarding step before finishing.",
        details: { reason: "onboarding_incomplete", missing_steps: ["schedule"] },
      }),
    );
    render(<OnboardingFlow />);
    const user = userEvent.setup();
    await screen.findByRole("heading", { name: "Where are you now?" });
    await user.click(confidence("Math", "3: Somewhat"));
    await user.click(confidence("Reading and Writing", "3: Somewhat"));
    await user.click(screen.getByRole("button", { name: "Finish" }));
    expect(await screen.findByRole("heading", { name: "Plan your study time" })).toBeTruthy();
    expect(replace).not.toHaveBeenCalled();
  });

  it("sends learners without a finished profile to Profile Setup", async () => {
    profileState.completed = false;
    mocked.fetchOnboarding.mockResolvedValue(onboarding());
    render(<OnboardingFlow />);
    await waitFor(() => expect(replace).toHaveBeenCalledWith("/setup/profile"));
  });

  it("moves on when onboarding is already finished", async () => {
    mocked.fetchOnboarding.mockResolvedValue(onboarding({ completed: true, next_step: null }));
    render(<OnboardingFlow />);
    await waitFor(() => expect(replace).toHaveBeenCalledWith("/app"));
  });

  it("shows an error state with retry", async () => {
    mocked.fetchOnboarding.mockRejectedValueOnce(new Error("offline"));
    mocked.fetchOnboarding.mockResolvedValueOnce(onboarding());
    render(<OnboardingFlow />);
    await userEvent.click(await screen.findByRole("button", { name: /try again/i }));
    expect(await screen.findByRole("heading", { name: "Set your SAT goal" })).toBeTruthy();
  });
});

describe("OnboardingGate", () => {
  it("renders the page once onboarding is finished", async () => {
    mocked.fetchOnboarding.mockResolvedValue(onboarding({ completed: true, next_step: null }));
    render(<OnboardingGate>dashboard</OnboardingGate>);
    expect(await screen.findByText("dashboard")).toBeTruthy();
    expect(replace).not.toHaveBeenCalled();
  });

  it("sends unfinished learners to onboarding", async () => {
    mocked.fetchOnboarding.mockResolvedValue(onboarding({ next_step: "schedule", goal }));
    render(<OnboardingGate>dashboard</OnboardingGate>);
    await waitFor(() => expect(replace).toHaveBeenCalledWith("/setup/onboarding"));
    expect(screen.queryByText("dashboard")).toBeNull();
  });

  it("shows an error state instead of the page when loading fails", async () => {
    mocked.fetchOnboarding.mockRejectedValue(new Error("offline"));
    render(<OnboardingGate>dashboard</OnboardingGate>);
    expect(await screen.findByText("We couldn't load your study goals")).toBeTruthy();
    expect(screen.queryByText("dashboard")).toBeNull();
  });
});
