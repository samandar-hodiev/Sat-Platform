import { describe, expect, it } from "vitest";

import {
  DEFAULT_PREFERENCES,
  cookieName,
  parsePreferences,
  readCookie,
  resolveTheme,
  scopeForPath,
  serializePreferences,
  themeAttributes,
} from "./theme";

describe("theme preferences", () => {
  it("defaults to neutral light mode (PRD §4.2)", () => {
    expect(parsePreferences(null)).toEqual(DEFAULT_PREFERENCES);
    expect(DEFAULT_PREFERENCES).toMatchObject({ mode: "light", accent: "neutral" });
  });

  it("round-trips through the cookie format", () => {
    const prefs = { mode: "dark", accent: "teal", density: "compact", motion: "reduced" } as const;
    expect(serializePreferences(prefs)).toBe("dark.teal.compact.reduced");
    expect(parsePreferences(serializePreferences(prefs))).toEqual(prefs);
  });

  it("falls back field by field on unknown or tampered values", () => {
    expect(parsePreferences("dark.<script>.compact")).toEqual({
      ...DEFAULT_PREFERENCES,
      mode: "dark",
      density: "compact",
    });
    expect(parsePreferences("garbage")).toEqual(DEFAULT_PREFERENCES);
  });

  it("keeps Learner and Owner preferences separate (PRD §21.3)", () => {
    expect(scopeForPath("/")).toBe("learner");
    expect(scopeForPath("/app/dashboard")).toBe("learner");
    expect(scopeForPath("/owner")).toBe("owner");
    expect(scopeForPath("/owner/learners")).toBe("owner");
    expect(scopeForPath("/ownership")).toBe("learner");
    expect(cookieName("learner")).not.toBe(cookieName("owner"));
  });

  it("reads one cookie out of a header", () => {
    const header = "a=1; sat_learner_appearance=dark.blue.comfortable.system; b=x%20y";
    expect(readCookie(header, "sat_learner_appearance")).toBe("dark.blue.comfortable.system");
    expect(readCookie(header, "b")).toBe("x y");
    expect(readCookie(header, "missing")).toBeNull();
    expect(readCookie("bad=%E0%A4%A", "bad")).toBeNull();
  });

  it("resolves system mode against the OS setting", () => {
    expect(resolveTheme("system", true)).toBe("dark");
    expect(resolveTheme("system", false)).toBe("light");
    expect(resolveTheme("light", true)).toBe("light");
    expect(themeAttributes({ ...DEFAULT_PREFERENCES, mode: "system" }, true)["data-theme"]).toBe(
      "dark",
    );
  });
});
