import { describe, expect, it } from "vitest";

import {
  latestExamDate,
  parseScore,
  previousScoreForApi,
  previousScoreForm,
  thisMonth,
  todayISO,
  validateExamDate,
  validatePreviousScore,
  validateStudyDays,
  validateTargetScore,
} from "./validation";

const now = new Date(2026, 9, 13, 9, 30); // 13 Oct 2026, local time

describe("onboarding validation", () => {
  it("formats local dates", () => {
    expect(todayISO(now)).toBe("2026-10-13");
    expect(latestExamDate(now)).toBe("2028-10-13");
    expect(thisMonth(now)).toBe("2026-10");
  });

  it("checks the target score", () => {
    expect(validateTargetScore("1400")).toBeUndefined();
    expect(validateTargetScore("400")).toBeUndefined();
    expect(validateTargetScore("")).toBe("Enter a score.");
    for (const bad of ["390", "1610", "1405", "14e2", "-400", "1400.0"]) {
      expect(validateTargetScore(bad)).toMatch(/from 400 to 1600/);
    }
    expect(parseScore(" 1200 ")).toBe(1200);
    expect(parseScore("12a")).toBeNaN();
  });

  it("checks the exam date, or accepts undecided", () => {
    expect(validateExamDate("", true, now)).toBeUndefined();
    expect(validateExamDate("", false, now)).toMatch(/haven't decided/);
    expect(validateExamDate("2026-10-13", false, now)).toBeUndefined();
    expect(validateExamDate("2026-10-12", false, now)).toMatch(/past/);
    expect(validateExamDate("2028-10-13", false, now)).toBeUndefined();
    expect(validateExamDate("2028-10-14", false, now)).toMatch(/2 years/);
  });

  it("requires a study day", () => {
    expect(validateStudyDays([])).toBeTruthy();
    expect(validateStudyDays([1])).toBeUndefined();
  });

  it("checks an earlier SAT result", () => {
    const base = { total: "1200", math: "", readingWriting: "", takenMonth: "" };
    expect(Object.values(validatePreviousScore(base, now)).filter(Boolean)).toHaveLength(0);
    expect(validatePreviousScore({ ...base, total: "" }, now).total).toBe("Enter a score.");
    expect(validatePreviousScore({ ...base, math: "600" }, now).readingWriting).toMatch(/both/);
    expect(validatePreviousScore({ ...base, readingWriting: "600" }, now).math).toMatch(/both/);
    expect(
      validatePreviousScore({ ...base, math: "700", readingWriting: "600" }, now).total,
    ).toMatch(/must equal/);
    expect(
      validatePreviousScore({ ...base, math: "900", readingWriting: "300" }, now).math,
    ).toMatch(/200 to 800/);
    expect(validatePreviousScore({ ...base, takenMonth: "2026-11" }, now).takenMonth).toMatch(
      /future/,
    );
    expect(validatePreviousScore({ ...base, takenMonth: "2015-12" }, now).takenMonth).toMatch(
      /2016/,
    );
  });

  it("converts the earlier result to and from the API shape", () => {
    const form = { total: "1210", math: "650", readingWriting: "560", takenMonth: "2026-05" };
    const api = previousScoreForApi(form);
    expect(api).toEqual({ total: 1210, math: 650, reading_writing: 560, taken_month: "2026-05" });
    expect(previousScoreForm(api)).toEqual(form);
    expect(
      previousScoreForApi({ total: "1000", math: "", readingWriting: "", takenMonth: "" }),
    ).toEqual({ total: 1000, math: null, reading_writing: null, taken_month: null });
  });
});
